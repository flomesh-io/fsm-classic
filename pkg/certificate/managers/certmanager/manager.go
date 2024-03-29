/*
 * MIT License
 *
 * Copyright (c) since 2021,  flomesh.io Authors.
 *
 * Permission is hereby granted, free of charge, to any person obtaining a copy
 * of this software and associated documentation files (the "Software"), to deal
 * in the Software without restriction, including without limitation the rights
 * to use, copy, modify, merge, publish, distribute, sublicense, and/or sell
 * copies of the Software, and to permit persons to whom the Software is
 * furnished to do so, subject to the following conditions:
 *
 * The above copyright notice and this permission notice shall be included in all
 * copies or substantial portions of the Software.
 *
 * THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
 * IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
 * FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
 * AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
 * LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
 * OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
 * SOFTWARE.
 */

package certmanager

import (
	"context"
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"crypto/x509/pkix"
	"fmt"
	"github.com/flomesh-io/fsm-classic/pkg/certificate"
	"github.com/flomesh-io/fsm-classic/pkg/certificate/utils"
	"github.com/flomesh-io/fsm-classic/pkg/commons"
	"github.com/flomesh-io/fsm-classic/pkg/config"
	"github.com/flomesh-io/fsm-classic/pkg/kube"
	certmgr "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	certmgrclient "github.com/jetstack/cert-manager/pkg/client/clientset/versioned"
	certmgrinformer "github.com/jetstack/cert-manager/pkg/client/informers/externalversions"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/klog/v2"
	"time"
)

func NewClient(k8sApi *kube.K8sAPI, mc *config.MeshConfig) *Client {
	namespace := mc.GetCaBundleNamespace()
	cmClient := certmgrclient.NewForConfigOrDie(k8sApi.Config)
	informerFactory := certmgrinformer.NewSharedInformerFactoryWithOptions(cmClient, time.Second*60, certmgrinformer.WithNamespace(namespace))

	certificates := informerFactory.Certmanager().V1().Certificates()
	certLister := certificates.Lister().Certificates(namespace)
	certInformer := certificates.Informer()

	certificateRequests := informerFactory.Certmanager().V1().CertificateRequests()
	crLister := certificateRequests.Lister().CertificateRequests(namespace)
	crInformer := certificateRequests.Informer()

	go certInformer.Run(wait.NeverStop)
	go crInformer.Run(wait.NeverStop)

	return &Client{
		ns:                       namespace,
		mc:                       mc,
		cmClient:                 cmClient,
		kubeClient:               k8sApi.Client,
		certificateRequestLister: crLister,
		certificateLister:        certLister,
	}
}

func NewManager(ca *certificate.Certificate, client *Client) (*CertManager, error) {
	return &CertManager{
		ca:     ca,
		client: client,
		issuerRef: cmmeta.ObjectReference{
			Kind:  IssuerKind,
			Name:  CAIssuerName,
			Group: IssuerGroup,
		},
		certificates: map[string]*certificate.Certificate{},
	}, nil
}

func NewRootCA(
	client *Client,
	cn string, validityPeriod time.Duration,
	country, locality, organization string,
) (*certificate.Certificate, error) {
	// create cert-manager SelfSigned issuer
	selfSigned, err := selfSignedIssuer(client)
	if err != nil {
		return nil, err
	}

	serialNumber, err := rand.Int(rand.Reader, certificate.SerialNumberLimit)
	if err != nil {
		return nil, fmt.Errorf("generate serial number: %s", err.Error())
	}

	ca := &certmgr.Certificate{
		ObjectMeta: metav1.ObjectMeta{
			Name:      CertManagerRootCAName,
			Namespace: client.ns,
		},
		Spec: certmgr.CertificateSpec{
			IsCA:       true,
			CommonName: cn,
			Duration: &metav1.Duration{
				Duration: validityPeriod,
			},
			Subject: &certmgr.X509Subject{
				Countries:     []string{country},
				Localities:    []string{locality},
				Organizations: []string{organization},
				SerialNumber:  serialNumber.String(),
			},
			PrivateKey: &certmgr.CertificatePrivateKey{
				Size:      2048,
				Encoding:  certmgr.PKCS8,
				Algorithm: certmgr.RSAKeyAlgorithm,
			},
			IssuerRef: cmmeta.ObjectReference{
				Kind:  selfSigned.Kind,
				Name:  selfSigned.Name,
				Group: selfSigned.GroupVersionKind().Group,
			},
			SecretName: client.mc.GetCaBundleName(),
		},
	}

	_, err = createCertManagerCertificate(client, ca)
	if err != nil {
		return nil, err
	}

	// create cert-manager CA issuer
	_, err = caIssuer(client)
	if err != nil {
		return nil, err
	}

	secret, err := client.kubeClient.CoreV1().
		Secrets(client.ns).
		Get(context.TODO(), client.mc.GetCaBundleName(), metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get cert-manager CA secret %s/%s: %s", client.ns, client.mc.GetCaBundleName(), err)
	}

	pemCACrt, ok := secret.Data[commons.RootCACertName]
	if !ok {
		klog.Errorf("Secret %s/%s doesn't have required %q data", client.ns, client.ns, client.mc.GetCaBundleName(), commons.RootCACertName)
		return nil, fmt.Errorf("invalid secret data for cert")
	}

	pemCAKey, ok := secret.Data[commons.TLSPrivateKeyName]
	if !ok {
		klog.Errorf("Secret %s/%s doesn't have required %q data", client.ns, client.ns, client.mc.GetCaBundleName(), commons.TLSPrivateKeyName)
		return nil, fmt.Errorf("invalid secret data for cert")
	}

	cert, err := utils.ConvertPEMCertToX509(pemCACrt)
	if err != nil {
		return nil, fmt.Errorf("failed to decoded root certificate: %s", err)
	}

	// NO private key for cert-manager generated cert
	return &certificate.Certificate{
		CommonName:   cert.Subject.CommonName,
		SerialNumber: cert.SerialNumber.String(),
		CA:           pemCACrt,
		CrtPEM:       pemCACrt,
		KeyPEM:       pemCAKey,
		Expiration:   cert.NotAfter,
	}, nil
}

func selfSignedIssuer(c *Client) (*certmgr.Issuer, error) {
	issuer := &certmgr.Issuer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SelfSignedIssuerName,
			Namespace: c.ns,
		},
		Spec: certmgr.IssuerSpec{
			IssuerConfig: certmgr.IssuerConfig{
				SelfSigned: &certmgr.SelfSignedIssuer{},
			},
		},
	}

	issuer, err := c.cmClient.CertmanagerV1().
		Issuers(c.ns).
		Create(context.Background(), issuer, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			// it's normal in case of race condition
			klog.V(2).Infof("Issuer %s/%s already exists.", c.ns, SelfSignedIssuerName)
			issuer, err = c.cmClient.CertmanagerV1().
				Issuers(c.ns).
				Get(context.Background(), SelfSignedIssuerName, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
		} else {
			klog.Errorf("create cert-manager issuer, %s", err.Error())
			return nil, err
		}
	}

	return issuer, nil
}

func caIssuer(c *Client) (*certmgr.Issuer, error) {
	issuer := &certmgr.Issuer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      CAIssuerName,
			Namespace: c.ns,
		},
		Spec: certmgr.IssuerSpec{
			IssuerConfig: certmgr.IssuerConfig{
				CA: &certmgr.CAIssuer{
					SecretName: c.mc.GetCaBundleName(),
				},
			},
		},
	}

	issuer, err := c.cmClient.CertmanagerV1().
		Issuers(c.ns).
		Create(context.Background(), issuer, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			// it's normal in case of race condition
			klog.V(2).Infof("Issuer %s/%s already exists.", c.ns, CAIssuerName)
			issuer, err = c.cmClient.CertmanagerV1().
				Issuers(c.ns).
				Get(context.Background(), CAIssuerName, metav1.GetOptions{})
			if err != nil {
				return nil, err
			}
		} else {
			klog.Errorf("create cert-manager issuer, %s", err.Error())
			return nil, err
		}
	}

	return issuer, nil
}

func createCertManagerCertificate(c *Client, cert *certmgr.Certificate) (*certmgr.Certificate, error) {
	cmCert, err := c.cmClient.CertmanagerV1().
		Certificates(c.ns).
		Create(context.Background(), cert, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			// it's normal in case of race condition
			klog.V(2).Infof("Certificate %s/%s already exists.", c.ns, CertManagerRootCAName)
		} else {
			klog.Errorf("create cert-manager certificate, %s", err.Error())
			return nil, err
		}
	}

	cmCert, err = waitingForCAIssued(c)
	if err != nil {
		return nil, err
	}

	return cmCert, nil
}

func waitingForCAIssued(c *Client) (*certmgr.Certificate, error) {
	var ca *certmgr.Certificate
	var err error

	// use lister to avoid invoke API server too frequently
	err = wait.PollImmediate(DefaultPollInterval, DefaultPollTimeout, func() (bool, error) {
		klog.V(2).Infof("Checking if CA %q is ready", CertManagerRootCAName)
		ca, err = c.certificateLister.Get(CertManagerRootCAName)
		if err != nil {
			if apierrors.IsNotFound(err) {
				// it takes time to sync, perhaps still not in the local store yet
				return false, nil
			} else {
				return false, err
			}
		}

		for _, condition := range ca.Status.Conditions {
			if condition.Type == certmgr.CertificateConditionReady &&
				condition.Status == cmmeta.ConditionTrue {
				// The cert has been issued, it takes time to issue a Certificate
				klog.V(2).Infof("CA %q is ready for use.", CertManagerRootCAName)
				return true, nil
			}
		}

		klog.V(2).Info("CA is not ready, waiting...")
		return false, nil
	})

	return ca, err
}

func (m CertManager) IssueCertificate(cn string, validityPeriod time.Duration, dnsNames []string) (*certificate.Certificate, error) {
	// TLS private key
	tlsKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("generate private key: %s", err.Error())
	}

	csr := &x509.CertificateRequest{
		Version:            3,
		SignatureAlgorithm: x509.SHA512WithRSA,
		PublicKeyAlgorithm: x509.RSA,
		Subject: pkix.Name{
			CommonName:   cn,
			Organization: []string{commons.DefaultCAOrganization},
		},
		DNSNames: dnsNames,
	}

	csrBytes, err := x509.CreateCertificateRequest(rand.Reader, csr, tlsKey)
	if err != nil {
		return nil, fmt.Errorf("error creating x509 certificate request: %s", err)
	}

	pemCSR, err := utils.CsrToPEM(csrBytes)
	if err != nil {
		return nil, fmt.Errorf("encode CSR: %s", err.Error())
	}

	certificateRequest := &certmgr.CertificateRequest{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: "cr-",
			Namespace:    m.client.ns,
		},
		Spec: certmgr.CertificateRequestSpec{
			Request: pemCSR,
			IsCA:    false,
			Duration: &metav1.Duration{
				Duration: validityPeriod,
			},
			Usages: []certmgr.KeyUsage{
				certmgr.UsageKeyEncipherment, certmgr.UsageDigitalSignature,
				certmgr.UsageClientAuth, certmgr.UsageServerAuth,
			},
			IssuerRef: m.issuerRef,
		},
	}

	cr, err := m.createCertManagerCertificateRequest(certificateRequest)
	if err != nil {
		return nil, err
	}

	// PEM encode key
	pemTlsKey, err := utils.RSAKeyToPEM(tlsKey)
	if err != nil {
		return nil, err
	}

	cert, err := utils.ConvertPEMCertToX509(cr.Status.Certificate)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err := m.client.cmClient.CertmanagerV1().
			CertificateRequests(m.client.ns).
			Delete(context.TODO(), cr.Name, metav1.DeleteOptions{}); err != nil {
			klog.Errorf("failed to delete CertificateRequest %s/%s", m.client.ns, cr.Name)
		}
	}()

	issuedCert := &certificate.Certificate{
		CommonName:   cert.Subject.CommonName,
		SerialNumber: cert.SerialNumber.String(),
		CA:           cr.Status.CA,
		CrtPEM:       cr.Status.Certificate,
		KeyPEM:       pemTlsKey,
		Expiration:   cert.NotAfter,
	}

	m.certificates[cert.Subject.CommonName] = issuedCert

	return issuedCert, nil
}

func (m CertManager) createCertManagerCertificateRequest(certificateRequest *certmgr.CertificateRequest) (*certmgr.CertificateRequest, error) {
	cr, err := m.client.cmClient.CertmanagerV1().
		CertificateRequests(m.client.ns).
		Create(context.Background(), certificateRequest, metav1.CreateOptions{})
	if err != nil {
		return nil, err
	}

	cr, err = m.waitingForCertificateIssued(cr.Name)
	if err != nil {
		return nil, err
	}

	return cr, nil
}

func (m CertManager) waitingForCertificateIssued(crName string) (*certmgr.CertificateRequest, error) {
	var cr *certmgr.CertificateRequest
	var err error

	// use lister to avoid invoke API server too frequently
	err = wait.PollImmediate(DefaultPollInterval, DefaultPollTimeout, func() (bool, error) {
		klog.V(3).Infof("Checking if CertificateRequest %q is ready", crName)
		cr, err = m.client.certificateRequestLister.Get(crName)
		if err != nil {
			if apierrors.IsNotFound(err) {
				// it takes time to sync, perhaps still not in the local store yet
				return false, nil
			} else {
				return false, err
			}
		}

		for _, condition := range cr.Status.Conditions {
			if condition.Type == certmgr.CertificateRequestConditionReady &&
				condition.Status == cmmeta.ConditionTrue &&
				cr.Status.Certificate != nil {
				// The cert has been issued, it takes time to fulfill a CertificateRequest
				klog.V(3).Infof("Certificate %q is ready for use.", crName)
				return true, nil
			}
		}

		klog.V(3).Info("Certificate is not ready, waiting...")
		return false, nil
	})

	return cr, err
}

func (m CertManager) GetCertificate(cn string) (*certificate.Certificate, error) {
	if cn == "" {
		return nil, fmt.Errorf("CN is empty")
	}

	cert, ok := m.certificates[cn]
	if !ok {
		return nil, fmt.Errorf("no certificate found for CN %q", cn)
	}

	return cert, nil
}

func (m CertManager) GetRootCertificate() (*certificate.Certificate, error) {
	return m.ca, nil
}
