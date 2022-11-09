package main

import (
	"flag"
	"fmt"
	"io"
	"net/http"
	"os"

	opentelekomcloud "github.com/opentelekomcloud/gophertelekomcloud"
	"github.com/opentelekomcloud/gophertelekomcloud/openstack"
	ipsec "github.com/opentelekomcloud/gophertelekomcloud/openstack/networking/v2/extensions/vpnaas/siteconnections"
	"k8s.io/klog/v2"
)

const (
	externalIpRawUrl string = "https://myexternalip.com/raw"
)

var (
	provider          *opentelekomcloud.ProviderClient
	ipSecConnectionId *string = flag.String("ipsec-connection-id", "", "open telekom cloud ipsec s2s connection id")
	region            *string = flag.String("region", "eu-de", "open telekom cloud region")
)

func main() {
	defer exit()

	flag.Usage = func() {
		fmt.Printf("Usage of %s:\n", os.Args[0])
		flag.PrintDefaults()
	}

	hostname, err := os.Hostname()
	if err != nil {
		klog.Fatalln(err)
	}

	klog.Infof("starting ipsec connection peer-address dynamic update from %s", hostname)

	if len(*ipSecConnectionId) < 1 {
		klog.Fatalln("no valid ipsec connection id; use argument \"--help\" to see usage")
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
	klog.InitFlags(nil)
	flag.Parse()

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
	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			klog.Errorln(err)
		}
	}(resp.Body)

	ipaddress, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}

	return string(ipaddress), nil
}

func updateIPSecConnection(id string, ipaddress string) error {
	klog.Infof("updating ipsec connection peer-address to %v...", ipaddress)

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

	klog.Infof("updated ipsec connection peer-address to %v...", ipaddress)

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
