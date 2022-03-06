#$vm_box = "flomesh/ubuntu20.04-k3s1.16"
#$vm_box = "flomesh/ubuntu20.04-k3s1.18"
#$vm_box = "flomesh/ubuntu20.04-k3s1.19"
$vm_box = "flomesh/ubuntu20.04-k3s1.20"
#$vm_box = "flomesh/ubuntu20.04-k3s1.21"

# $k3s_ip = "192.168.77.100"

Vagrant.configure("2") do |config|
	config.vm.define "fsm", primary: true do |k3s|
		k3s.vm.box = $vm_box

		k3s.vm.box_check_update = false
	  
		k3s.vm.hostname = "fsm"
	  
		k3s.vm.network "private_network", type: "dhcp"

		k3s.vm.provider "virtualbox" do |vb|
			vb.name = "fsm"
			vb.memory = 8192 #$vm_memory
			vb.cpus = 2 #$vm_cpus
			vb.gui = false
		end
	end
end
