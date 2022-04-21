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
	"github.com/flomesh-io/traffic-guru/pkg/certificate"
	"github.com/flomesh-io/traffic-guru/pkg/certificate/utils"
	"github.com/flomesh-io/traffic-guru/pkg/commons"
	certmgr "github.com/jetstack/cert-manager/pkg/apis/certmanager/v1"
	cmmeta "github.com/jetstack/cert-manager/pkg/apis/meta/v1"
	certmgrclient "github.com/jetstack/cert-manager/pkg/client/clientset/versioned"
	certmgrinformer "github.com/jetstack/cert-manager/pkg/client/informers/externalversions"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
	"k8s.io/klog/v2"
	"time"
)

func NewManager(ca *certificate.Certificate, client certmgrclient.Interface) (*CertManager, error) {
	informerFactory := certmgrinformer.NewSharedInformerFactory(client, time.Second*60)

	certificates := informerFactory.Certmanager().V1().Certificates()
	certLister := certificates.Lister().Certificates(commons.DefaultFlomeshNamespace)
	certInformer := certificates.Informer()

	certificateRequests := informerFactory.Certmanager().V1().CertificateRequests()
	crLister := certificateRequests.Lister().CertificateRequests(commons.DefaultFlomeshNamespace)
	crInformer := certificateRequests.Informer()

	go certInformer.Run(wait.NeverStop)
	go crInformer.Run(wait.NeverStop)

	return &CertManager{
		ca:                       ca,
		client:                   client,
		certificateRequestLister: crLister,
		certificateLister:        certLister,
		issuerRef: cmmeta.ObjectReference{
			Kind:  IssuerKind,
			Name:  CAIssuerName,
			Group: IssuerGroup,
		},
	}, nil
}

func NewRootCA(
	client certmgrclient.Interface,
	kubeClient kubernetes.Interface,
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
			Namespace: commons.DefaultFlomeshNamespace,
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
			SecretName: commons.DefaultCABundleName,
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

	secret, err := kubeClient.CoreV1().
		Secrets(commons.DefaultFlomeshNamespace).
		Get(context.TODO(), commons.DefaultCABundleName, metav1.GetOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to get cert-manager CA secret %s/%s: %s", commons.DefaultFlomeshNamespace, commons.DefaultCABundleName, err)
	}

	pemCACrt, ok := secret.Data[commons.RootCACertName]
	if !ok {
		klog.Errorf("Secret %s/%s doesn't have required %q data", commons.DefaultFlomeshNamespace, commons.DefaultCABundleName, commons.RootCACertName)
		return nil, fmt.Errorf("invalid secret data for cert")
	}

	pemCAKey, ok := secret.Data[commons.TLSPrivateKeyName]
	if !ok {
		klog.Errorf("Secret %s/%s doesn't have required %q data", commons.DefaultFlomeshNamespace, commons.DefaultCABundleName, commons.TLSPrivateKeyName)
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
		RootCA:       pemCACrt,
		CrtPEM:       pemCACrt,
		KeyPEM:       pemCAKey,
		Expiration:   cert.NotAfter,
	}, nil
}

func selfSignedIssuer(client certmgrclient.Interface) (*certmgr.Issuer, error) {
	issuer := &certmgr.Issuer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      SelfSignedIssuerName,
			Namespace: commons.DefaultFlomeshNamespace,
		},
		Spec: certmgr.IssuerSpec{
			IssuerConfig: certmgr.IssuerConfig{
				SelfSigned: &certmgr.SelfSignedIssuer{},
			},
		},
	}

	issuer, err := client.CertmanagerV1().
		Issuers(commons.DefaultFlomeshNamespace).
		Create(context.Background(), issuer, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			// it's normal in case of race condition
			klog.V(2).Infof("Issuer %s/%s already exists.", commons.DefaultFlomeshNamespace, SelfSignedIssuerName)
			issuer, err = client.CertmanagerV1().
				Issuers(commons.DefaultFlomeshNamespace).
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

func caIssuer(client certmgrclient.Interface) (*certmgr.Issuer, error) {
	issuer := &certmgr.Issuer{
		ObjectMeta: metav1.ObjectMeta{
			Name:      CAIssuerName,
			Namespace: commons.DefaultFlomeshNamespace,
		},
		Spec: certmgr.IssuerSpec{
			IssuerConfig: certmgr.IssuerConfig{
				CA: &certmgr.CAIssuer{
					SecretName: commons.DefaultCABundleName,
				},
			},
		},
	}

	issuer, err := client.CertmanagerV1().
		Issuers(commons.DefaultFlomeshNamespace).
		Create(context.Background(), issuer, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			// it's normal in case of race condition
			klog.V(2).Infof("Issuer %s/%s already exists.", commons.DefaultFlomeshNamespace, CAIssuerName)
			issuer, err = client.CertmanagerV1().
				Issuers(commons.DefaultFlomeshNamespace).
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

func createCertManagerCertificate(client certmgrclient.Interface, cert *certmgr.Certificate) (*certmgr.Certificate, error) {
	certificate, err := client.CertmanagerV1().
		Certificates(commons.DefaultFlomeshNamespace).
		Create(context.Background(), cert, metav1.CreateOptions{})
	if err != nil {
		if apierrors.IsAlreadyExists(err) {
			// it's normal in case of race condition
			klog.V(2).Infof("Certificate %s/%s already exists.", commons.DefaultFlomeshNamespace, CertManagerRootCAName)
		} else {
			klog.Errorf("create cert-manager certificate, %s", err.Error())
			return nil, err
		}
	}

	certificate, err = waitingForCAIssued(client)
	if err != nil {
		return nil, err
	}

	return certificate, nil
}

func waitingForCAIssued(client certmgrclient.Interface) (*certmgr.Certificate, error) {
	var ca *certmgr.Certificate
	var err error

	err = wait.PollImmediate(3*time.Second, 30*time.Second, func() (bool, error) {
		ca, err = client.CertmanagerV1().
			Certificates(commons.DefaultFlomeshNamespace).
			Get(context.Background(), CertManagerRootCAName, metav1.GetOptions{})

		if err != nil {
			return false, err
		}

		for _, condition := range ca.Status.Conditions {
			if condition.Type == certmgr.CertificateConditionReady && condition.Status == cmmeta.ConditionTrue {
				return true, nil
			}
		}

		klog.V(3).Info("CA is not ready, waiting...")
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
			Namespace:    commons.DefaultFlomeshNamespace,
			Annotations: map[string]string{
				certmgr.CertificateRequestPrivateKeyAnnotationKey: commons.DefaultCABundleName,
			},
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
		if err := m.client.CertmanagerV1().
			CertificateRequests(commons.DefaultFlomeshNamespace).
			Delete(context.TODO(), cr.Name, metav1.DeleteOptions{}); err != nil {
			klog.Errorf("failed to delete CertificateRequest %s/%s", commons.DefaultFlomeshNamespace, cr.Name)
		}
	}()

	return &certificate.Certificate{
		CommonName:   cert.Subject.CommonName,
		SerialNumber: cert.SerialNumber.String(),
		RootCA:       cr.Status.CA,
		CrtPEM:       cr.Status.Certificate,
		KeyPEM:       pemTlsKey,
		Expiration:   cert.NotAfter,
	}, nil
}

func (m CertManager) createCertManagerCertificateRequest(certificateRequest *certmgr.CertificateRequest) (*certmgr.CertificateRequest, error) {
	cr, err := m.client.CertmanagerV1().
		CertificateRequests(commons.DefaultFlomeshNamespace).
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
	err = wait.PollImmediate(1*time.Second, 60*time.Second, func() (bool, error) {
		klog.V(5).Infof("Checking if CertificateRequest %q is ready", crName)
		cr, err = m.certificateRequestLister.Get(crName)
		if err != nil {
			if apierrors.IsNotFound(err) {
				// it takes time to sync, perhaps still not in the local store yet
				return false, nil
			} else {
				return false, err
			}
		}
		// The cert has been issued
		for _, condition := range cr.Status.Conditions {
			if condition.Type == certmgr.CertificateRequestConditionReady &&
				condition.Status == cmmeta.ConditionTrue {
				if cr.Status.Certificate != nil {
					klog.V(5).Infof("Certificate is ready for use.")
					return true, nil
				}
			}
		}

		return false, nil
	})

	return cr, err
}

func (m CertManager) GetCertificate(cn string) (*certificate.Certificate, error) {
	//TODO implement me
	panic("implement me")
}

func (m CertManager) GetRootCertificate() (*certificate.Certificate, error) {
	return m.ca, nil
}
