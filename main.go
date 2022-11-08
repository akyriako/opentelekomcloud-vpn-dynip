package main

import (
	"flag"
	"io"
	"net/http"

	opentelekomcloud "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack"
	ipsec "github.com/opentelekomcloud/gophertelekomcloud/openstack/networking/v2/extensions/vpnaas/siteconnections"
	"k8s.io/klog/v2"
)

const (
	externalIpRawUrl string = "http://myexternalip.com/raw"
)

var (
	provider          *opentelekomcloud.ProviderClient
	ipSecConnectionId *string = flag.String("ipsec-connection-id", "", "ipsec s2s connection id")
	region            *string = flag.String("region", "eu-de", "open telekom cloud region")
)

func main() {
	defer exit()

	klog.InitFlags(nil)
	flag.Parse()

	klog.Infoln("starting ipsec connection peer-address dynamic update")

	if len(*ipSecConnectionId) < 1 {
		klog.Fatalln("no valid ipsec connection id")
	}

	externalIp, err := getExternalIP()
	if err != nil {
		klog.Fatalln(err)
	}

	klog.Infof("current external ip: %v", externalIp)

	connection, err := getIPSecConnection(*ipSecConnectionId)
	if err != nil {
		klog.Fatalln(err)
	}

	klog.Infof("current ipsec connection peer-address: %v", connection.PeerAddress)

	if connection.PeerAddress != externalIp {
		err := updateIPSecConnection(*ipSecConnectionId, externalIp)
		if err != nil {
			klog.Fatalln(err)
		}
	} else {
		klog.Infoln("skip update, no ip address change...")
	}
}

func init() {
	klog.Infoln("initializing openstack provider client")

	authOptions, err := openstack.AuthOptionsFromEnv()
	if err != nil {
		klog.Fatalln(err)
	}

	providerClient, err := openstack.AuthenticatedClient(authOptions)
	if err != nil {
		klog.Fatalln(err)
	}

	provider = providerClient
	klog.Infoln("initialized openstack provider client")
}

func exit() {
	exitCode := 10
	klog.Infoln("flush logs & exit...")
	klog.FlushAndExit(klog.ExitFlushTimeout, exitCode)
}

func getExternalIP() (string, error) {
	resp, err := http.Get(externalIpRawUrl)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()

	ipaddress, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(ipaddress), nil
}

func updateIPSecConnection(id string, ipaddress string) error {
	klog.Infoln("updating ipsec connection peer-address...")

	client, err := openstack.NewNetworkV2(provider, opentelekomcloud.EndpointOpts{
		Region: *region,
	})
	if err != nil {
		return err
	}

	updateOpts := ipsec.UpdateOpts{
		PeerAddress: ipaddress,
	}
	_, err = ipsec.Update(client, id, updateOpts).Extract()
	if err != nil {
		return err
	}

	klog.Infoln("updated ipsec connection peer-address...")

	return nil
}

func getIPSecConnection(id string) (*ipsec.Connection, error) {
	client, err := openstack.NewNetworkV2(provider, opentelekomcloud.EndpointOpts{
		Region: *region,
	})
	if err != nil {
		return nil, err
	}

	connection, err := ipsec.Get(client, id).Extract()
	if err != nil {
		return nil, err
	}

	return connection, nil
}
