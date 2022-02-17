//go:build !windows
// +build !windows

package net_test

import (
	"errors"
	"fmt"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	. "github.com/cloudfoundry/bosh-agent/platform/net"
	fakearp "github.com/cloudfoundry/bosh-agent/platform/net/arp/fakes"
	boship "github.com/cloudfoundry/bosh-agent/platform/net/ip"
	fakeip "github.com/cloudfoundry/bosh-agent/platform/net/ip/fakes"
	"github.com/cloudfoundry/bosh-agent/platform/net/netfakes"
	boshsettings "github.com/cloudfoundry/bosh-agent/settings"
	boshlog "github.com/cloudfoundry/bosh-utils/logger"
	fakesys "github.com/cloudfoundry/bosh-utils/system/fakes"
)

var _ = Describe("centosNetManager", describeCentosNetManager)

func describeCentosNetManager() {
	var (
		fs                            *fakesys.FakeFileSystem
		cmdRunner                     *fakesys.FakeCmdRunner
		ipResolver                    *fakeip.FakeResolver
		interfaceAddrsProvider        *fakeip.FakeInterfaceAddressesProvider
		addressBroadcaster            *fakearp.FakeAddressBroadcaster
		netManager                    Manager
		interfaceConfigurationCreator InterfaceConfigurationCreator
		fakeMACAddressDetector        *netfakes.FakeMACAddressDetector
	)

	stubInterfaces := func(physicalInterfaces map[string]boshsettings.Network) {
		addresses := map[string]string{}
		for iface, networkSettings := range physicalInterfaces {
			addresses[networkSettings.Mac] = iface
		}

		fakeMACAddressDetector.DetectMacAddressesReturns(addresses, nil)
	}

	BeforeEach(func() {
		fs = fakesys.NewFakeFileSystem()
		cmdRunner = fakesys.NewFakeCmdRunner()
		ipResolver = &fakeip.FakeResolver{}
		logger := boshlog.NewLogger(boshlog.LevelNone)
		fakeMACAddressDetector = &netfakes.FakeMACAddressDetector{}
		interfaceConfigurationCreator = NewInterfaceConfigurationCreator(logger)
		interfaceAddrsProvider = &fakeip.FakeInterfaceAddressesProvider{}
		dnsValidator := NewDNSValidator(fs)
		addressBroadcaster = &fakearp.FakeAddressBroadcaster{}
		netManager = NewCentosNetManager(
			fs,
			cmdRunner,
			ipResolver,
			fakeMACAddressDetector,
			interfaceConfigurationCreator,
			interfaceAddrsProvider,
			dnsValidator,
			addressBroadcaster,
			logger,
		)
	})

	Describe("SetupNetworking", func() {
		var (
			dhcpNetwork                           boshsettings.Network
			staticNetwork                         boshsettings.Network
			expectedNetworkConfigurationForStatic string
			expectedNetworkConfigurationForDHCP   string
			expectedDhclientConfiguration         string
		)

		BeforeEach(func() {
			dhcpNetwork = boshsettings.Network{
				Type:    "dynamic",
				Default: []string{"dns"},
				DNS:     []string{"8.8.8.8", "9.9.9.9"},
				Mac:     "fake-dhcp-mac-address",
			}
			staticNetwork = boshsettings.Network{
				Type:    "manual",
				IP:      "1.2.3.4",
				Netmask: "255.255.255.0",
				Gateway: "3.4.5.6",
				Mac:     "fake-static-mac-address",
			}
			interfaceAddrsProvider.GetInterfaceAddresses = []boship.InterfaceAddress{
				boship.NewSimpleInterfaceAddress("ethstatic", "1.2.3.4"),
			}
			fs.WriteFileString("/etc/resolv.conf", `
nameserver 8.8.8.8
nameserver 9.9.9.9
`)

			expectedNetworkConfigurationForStatic = `DEVICE=ethstatic
BOOTPROTO=static
IPADDR=1.2.3.4
NETMASK=255.255.255.0
BROADCAST=1.2.3.255
ONBOOT=yes
PEERDNS=no
DNS1=8.8.8.8
DNS2=9.9.9.9
`

			expectedNetworkConfigurationForDHCP = `DEVICE=ethdhcp
BOOTPROTO=dhcp
ONBOOT=yes
PEERDNS=yes
`

			expectedDhclientConfiguration = `# Generated by bosh-agent

option rfc3442-classless-static-routes code 121 = array of unsigned integer 8;

send host-name "<hostname>";

request subnet-mask, broadcast-address, time-offset, routers,
	domain-name, domain-name-servers, domain-search, host-name,
	netbios-name-servers, netbios-scope, interface-mtu,
	rfc3442-classless-static-routes, ntp-servers;

prepend domain-name-servers 8.8.8.8, 9.9.9.9;
`
		})

		It("writes a network script for static and dynamic interfaces", func() {
			stubInterfaces(map[string]boshsettings.Network{
				"ethdhcp":   dhcpNetwork,
				"ethstatic": staticNetwork,
			})

			err := netManager.SetupNetworking(boshsettings.Networks{"dhcp-network": dhcpNetwork, "static-network": staticNetwork}, nil)
			Expect(err).ToNot(HaveOccurred())

			staticConfig := fs.GetFileTestStat("/etc/sysconfig/network-scripts/ifcfg-ethstatic")
			Expect(staticConfig).ToNot(BeNil())
			Expect(staticConfig.StringContents()).To(Equal(expectedNetworkConfigurationForStatic))

			dhcpConfig := fs.GetFileTestStat("/etc/sysconfig/network-scripts/ifcfg-ethdhcp")
			Expect(dhcpConfig).ToNot(BeNil())
			Expect(dhcpConfig.StringContents()).To(Equal(expectedNetworkConfigurationForDHCP))
		})

		It("doesn't write /etc/resolv.conf with dns servers", func() {
			stubInterfaces(map[string]boshsettings.Network{
				"ethdhcp":   dhcpNetwork,
				"ethstatic": staticNetwork,
			})

			err := netManager.SetupNetworking(boshsettings.Networks{"dhcp-network": dhcpNetwork, "static-network": staticNetwork}, nil)
			Expect(err).ToNot(HaveOccurred())

			resolvConf := fs.GetFileTestStat("/etc/resolv.conf")
			Expect(resolvConf).ToNot(BeNil())
			Expect(resolvConf.StringContents()).To(Equal(`
nameserver 8.8.8.8
nameserver 9.9.9.9
`))
		})

		It("doesn't write /etc/resolv.conf if there are no dns servers", func() {
			dhcpNetworkWithoutDNS := boshsettings.Network{
				Type: "dynamic",
				Mac:  "fake-dhcp-mac-address",
			}

			stubInterfaces(map[string]boshsettings.Network{
				"ethdhcp": dhcpNetworkWithoutDNS,
			})

			err := netManager.SetupNetworking(boshsettings.Networks{"dhcp-network": dhcpNetworkWithoutDNS}, nil)
			Expect(err).ToNot(HaveOccurred())

			resolvConf := fs.GetFileTestStat("/etc/resolv.conf")
			Expect(resolvConf).ToNot(BeNil())
			Expect(resolvConf.StringContents()).To(Equal(`
nameserver 8.8.8.8
nameserver 9.9.9.9
`))

		})

		It("returns errors from writing the network configuration", func() {
			stubInterfaces(map[string]boshsettings.Network{
				"dhcp":   dhcpNetwork,
				"static": staticNetwork,
			})
			fs.WriteFileError = errors.New("fs-write-file-error")
			err := netManager.SetupNetworking(boshsettings.Networks{"dhcp-network": dhcpNetwork, "static-network": staticNetwork}, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("fs-write-file-error"))
		})

		It("returns errors when it can't create network interface configurations", func() {
			stubInterfaces(map[string]boshsettings.Network{
				"ethstatic": staticNetwork,
			})

			staticNetwork.Netmask = "not an ip" //will cause InterfaceConfigurationCreator to fail
			err := netManager.SetupNetworking(boshsettings.Networks{"static-network": staticNetwork}, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Creating interface configurations"))
		})

		It("writes a dhcp configuration if there are dhcp networks", func() {
			stubInterfaces(map[string]boshsettings.Network{
				"ethdhcp":   dhcpNetwork,
				"ethstatic": staticNetwork,
			})

			err := netManager.SetupNetworking(boshsettings.Networks{"dhcp-network": dhcpNetwork, "static-network": staticNetwork}, nil)
			Expect(err).ToNot(HaveOccurred())

			dhcpConfig := fs.GetFileTestStat("/etc/dhcp/dhclient.conf")
			Expect(dhcpConfig).ToNot(BeNil())
			Expect(dhcpConfig.StringContents()).To(Equal(expectedDhclientConfiguration))

			dhcpConfigSymlink := fs.GetFileTestStat("/etc/dhcp/dhclient-ethdhcp.conf")
			Expect(dhcpConfigSymlink).ToNot(BeNil())
			Expect(dhcpConfigSymlink.SymlinkTarget).To(Equal("/etc/dhcp/dhclient.conf"))
		})

		It("writes a dhcp configuration without prepended dns servers if there are no dns servers specified", func() {
			dhcpNetworkWithoutDNS := boshsettings.Network{
				Type: "dynamic",
				Mac:  "fake-dhcp-mac-address",
			}

			stubInterfaces(map[string]boshsettings.Network{
				"ethdhcp": dhcpNetwork,
			})

			err := netManager.SetupNetworking(boshsettings.Networks{"dhcp-network": dhcpNetworkWithoutDNS}, nil)
			Expect(err).ToNot(HaveOccurred())

			dhcpConfig := fs.GetFileTestStat("/etc/dhcp/dhclient.conf")
			Expect(dhcpConfig).ToNot(BeNil())
			Expect(dhcpConfig.StringContents()).To(Equal(`# Generated by bosh-agent

option rfc3442-classless-static-routes code 121 = array of unsigned integer 8;

send host-name "<hostname>";

request subnet-mask, broadcast-address, time-offset, routers,
	domain-name, domain-name-servers, domain-search, host-name,
	netbios-name-servers, netbios-scope, interface-mtu,
	rfc3442-classless-static-routes, ntp-servers;

`))
			dhcpConfigSymlink := fs.GetFileTestStat("/etc/dhcp/dhclient-ethdhcp.conf")
			Expect(dhcpConfigSymlink).ToNot(BeNil())
			Expect(dhcpConfigSymlink.SymlinkTarget).To(Equal("/etc/dhcp/dhclient.conf"))
		})

		It("returns an error if it can't write a dhcp configuration", func() {
			stubInterfaces(map[string]boshsettings.Network{
				"ethdhcp":   dhcpNetwork,
				"ethstatic": staticNetwork,
			})

			fs.WriteFileErrors["/etc/dhcp/dhclient.conf"] = errors.New("dhclient.conf-write-error")

			err := netManager.SetupNetworking(boshsettings.Networks{"dhcp-network": dhcpNetwork, "static-network": staticNetwork}, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("dhclient.conf-write-error"))
		})

		It("returns an error if it can't symlink a dhcp configuration", func() {
			stubInterfaces(map[string]boshsettings.Network{
				"ethdhcp":   dhcpNetwork,
				"ethstatic": staticNetwork,
			})

			fs.SymlinkError = errors.New("dhclient-ethdhcp.conf-symlink-error")

			err := netManager.SetupNetworking(boshsettings.Networks{"dhcp-network": dhcpNetwork, "static-network": staticNetwork}, nil)
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("dhclient-ethdhcp.conf-symlink-error"))
		})

		It("doesn't write a dhcp configuration if there are no dhcp networks", func() {
			stubInterfaces(map[string]boshsettings.Network{
				"ethstatic": staticNetwork,
			})

			err := netManager.SetupNetworking(boshsettings.Networks{"static-network": staticNetwork}, nil)
			Expect(err).ToNot(HaveOccurred())

			dhcpConfig := fs.GetFileTestStat("/etc/dhcp/dhclient-ethdhcp.conf")
			Expect(dhcpConfig).To(BeNil())
		})

		It("restarts the networks if any ifconfig file changes", func() {
			changingStaticNetwork := boshsettings.Network{
				Type:    "manual",
				IP:      "1.2.3.5",
				Netmask: "255.255.255.0",
				Gateway: "3.4.5.6",
				Mac:     "ethstatict-that-changes",
			}

			stubInterfaces(map[string]boshsettings.Network{
				"ethdhcp":                dhcpNetwork,
				"ethstatic-that-changes": changingStaticNetwork,
				"ethstatic":              staticNetwork,
			})
			interfaceAddrsProvider.GetInterfaceAddresses = []boship.InterfaceAddress{
				boship.NewSimpleInterfaceAddress("ethstatic", "1.2.3.4"),
				boship.NewSimpleInterfaceAddress("ethstatic-that-changes", "1.2.3.5"),
			}

			fs.WriteFileString("/etc/sysconfig/network-scripts/ifcfg-ethstatic", expectedNetworkConfigurationForStatic)
			fs.WriteFileString("/etc/dhcp/dhclient.conf", expectedDhclientConfiguration)

			err := netManager.SetupNetworking(boshsettings.Networks{
				"dhcp-network":            dhcpNetwork,
				"changing-static-network": changingStaticNetwork,
				"static-network":          staticNetwork,
			},
				nil)
			Expect(err).ToNot(HaveOccurred())

			Expect(len(cmdRunner.RunCommands)).To(Equal(1))
			Expect(cmdRunner.RunCommands[0]).To(Equal([]string{"service", "network", "restart"}))
		})

		It("doesn't restart the networks if ifcfg and /etc/dhcp/dhclient.conf don't change", func() {
			stubInterfaces(map[string]boshsettings.Network{
				"ethdhcp":   dhcpNetwork,
				"ethstatic": staticNetwork,
			})

			fs.WriteFileString("/etc/sysconfig/network-scripts/ifcfg-ethstatic", expectedNetworkConfigurationForStatic)
			fs.WriteFileString("/etc/sysconfig/network-scripts/ifcfg-ethdhcp", expectedNetworkConfigurationForDHCP)
			fs.WriteFileString("/etc/dhcp/dhclient.conf", expectedDhclientConfiguration)

			err := netManager.SetupNetworking(boshsettings.Networks{"dhcp-network": dhcpNetwork, "static-network": staticNetwork}, nil)
			Expect(err).ToNot(HaveOccurred())

			networkConfig := fs.GetFileTestStat("/etc/sysconfig/network-scripts/ifcfg-ethstatic")
			Expect(networkConfig).ToNot(BeNil())
			Expect(networkConfig.StringContents()).To(Equal(expectedNetworkConfigurationForStatic))

			dhcpConfig := fs.GetFileTestStat("/etc/dhcp/dhclient.conf")
			Expect(dhcpConfig.StringContents()).To(Equal(expectedDhclientConfiguration))

			Expect(len(cmdRunner.RunCommands)).To(Equal(0))
		})

		It("restarts the networks if /etc/dhcp/dhclient.conf changes", func() {
			stubInterfaces(map[string]boshsettings.Network{
				"ethdhcp":   dhcpNetwork,
				"ethstatic": staticNetwork,
			})

			fs.WriteFileString("/etc/sysconfig/network-scripts/ifcfg-ethstatic", expectedNetworkConfigurationForStatic)

			err := netManager.SetupNetworking(boshsettings.Networks{"dhcp-network": dhcpNetwork, "static-network": staticNetwork}, nil)
			Expect(err).ToNot(HaveOccurred())

			networkConfig := fs.GetFileTestStat("/etc/sysconfig/network-scripts/ifcfg-ethstatic")
			Expect(networkConfig).ToNot(BeNil())
			Expect(networkConfig.StringContents()).To(Equal(expectedNetworkConfigurationForStatic))

			Expect(len(cmdRunner.RunCommands)).To(Equal(1))
			Expect(cmdRunner.RunCommands[0]).To(Equal([]string{"service", "network", "restart"}))
		})

		It("broadcasts MAC addresses for all interfaces", func() {
			stubInterfaces(map[string]boshsettings.Network{
				"ethdhcp":   dhcpNetwork,
				"ethstatic": staticNetwork,
			})

			errCh := make(chan error)
			err := netManager.SetupNetworking(boshsettings.Networks{"dhcp-network": dhcpNetwork, "static-network": staticNetwork}, errCh)
			Expect(err).ToNot(HaveOccurred())

			broadcastErr := <-errCh // wait for all arpings
			Expect(broadcastErr).ToNot(HaveOccurred())

			Expect(addressBroadcaster.Value()).To(Equal([]boship.InterfaceAddress{
				boship.NewSimpleInterfaceAddress("ethstatic", "1.2.3.4"),
				boship.NewResolvingInterfaceAddress("ethdhcp", ipResolver),
			}))

		})

		It("skips vip networks", func() {
			stubInterfaces(map[string]boshsettings.Network{
				"ethdhcp":   dhcpNetwork,
				"ethstatic": staticNetwork,
			})

			vipNetwork := boshsettings.Network{
				Type:    "vip",
				Default: []string{"dns"},
				DNS:     []string{"4.4.4.4", "5.5.5.5"},
				Mac:     "fake-vip-mac-address",
				IP:      "9.8.7.6",
			}

			err := netManager.SetupNetworking(boshsettings.Networks{
				"dhcp-network":   dhcpNetwork,
				"static-network": staticNetwork,
				"vip-network":    vipNetwork,
			}, nil)
			Expect(err).ToNot(HaveOccurred())

			networkConfig := fs.GetFileTestStat("/etc/sysconfig/network-scripts/ifcfg-ethstatic")
			Expect(networkConfig).ToNot(BeNil())
			Expect(networkConfig.StringContents()).To(Equal(expectedNetworkConfigurationForStatic))
		})

		It("doesn't use vip networks dns", func() {
			stubInterfaces(map[string]boshsettings.Network{
				"ethstatic": staticNetwork,
			})

			vipNetwork := boshsettings.Network{
				Type:    "vip",
				Default: []string{"dns"},
				DNS:     []string{"4.4.4.4", "5.5.5.5"},
				Mac:     "fake-vip-mac-address",
				IP:      "9.8.7.6",
			}

			err := netManager.SetupNetworking(boshsettings.Networks{
				"vip-network":    vipNetwork,
				"static-network": staticNetwork,
			}, nil)
			Expect(err).ToNot(HaveOccurred())

			networkConfig := fs.GetFileTestStat("/etc/sysconfig/network-scripts/ifcfg-ethstatic")
			Expect(networkConfig).ToNot(BeNil())
			Expect(networkConfig.StringContents()).ToNot(ContainSubstring("4.4.4.4"))
			Expect(networkConfig.StringContents()).ToNot(ContainSubstring("5.5.5.5"))
		})

		Context("when manual networks were not configured with proper IP addresses", func() {
			BeforeEach(func() {
				interfaceAddrsProvider.GetInterfaceAddresses = []boship.InterfaceAddress{
					boship.NewSimpleInterfaceAddress("ethstatic", "1.2.3.5"),
				}
			})

			It("fails", func() {
				stubInterfaces(map[string]boshsettings.Network{
					"ethstatic": staticNetwork,
				})

				errCh := make(chan error)
				err := netManager.SetupNetworking(boshsettings.Networks{"static-network": staticNetwork}, errCh)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Validating static network configuration"))
			})
		})

		Context("when dns is not properly configured", func() {
			BeforeEach(func() {
				fs.WriteFileString("/etc/resolv.conf", "")
			})

			It("fails", func() {
				staticNetwork = boshsettings.Network{
					Type:    "manual",
					IP:      "1.2.3.4",
					Default: []string{"dns"},
					DNS:     []string{"8.8.8.8"},
					Netmask: "255.255.255.0",
					Gateway: "3.4.5.6",
					Mac:     "fake-static-mac-address",
				}

				stubInterfaces(map[string]boshsettings.Network{
					"ethstatic": staticNetwork,
				})

				errCh := make(chan error)
				err := netManager.SetupNetworking(boshsettings.Networks{"static-network": staticNetwork}, errCh)
				Expect(err).To(HaveOccurred())
				Expect(err.Error()).To(ContainSubstring("Validating dns configuration"))
			})
		})

		Context("when no MAC address is provided in the settings", func() {
			var staticNetworkWithoutMAC boshsettings.Network

			BeforeEach(func() {
				staticNetworkWithoutMAC = boshsettings.Network{
					Type:    "manual",
					IP:      "1.2.3.4",
					Netmask: "255.255.255.0",
					Gateway: "3.4.5.6",
					DNS:     []string{"8.8.8.8", "9.9.9.9"},
					Default: []string{"dns"},
				}
			})

			It("configures network for single device", func() {
				stubInterfaces(
					map[string]boshsettings.Network{
						"ethstatic": staticNetwork,
					},
				)

				err := netManager.SetupNetworking(boshsettings.Networks{
					"static-network": staticNetworkWithoutMAC,
				}, nil)
				Expect(err).ToNot(HaveOccurred())

				networkConfig := fs.GetFileTestStat("/etc/sysconfig/network-scripts/ifcfg-ethstatic")
				Expect(networkConfig).ToNot(BeNil())
				Expect(networkConfig.StringContents()).To(Equal(expectedNetworkConfigurationForStatic))
			})
		})

		It("configures gateway, broadcast and dns for default network only", func() {
			staticNetwork = boshsettings.Network{
				Type:    "manual",
				IP:      "1.2.3.4",
				Netmask: "255.255.255.0",
				Gateway: "3.4.5.6",
				Mac:     "fake-static-mac-address",
			}
			secondStaticNetwork := boshsettings.Network{
				Type:    "manual",
				IP:      "5.6.7.8",
				Netmask: "255.255.255.0",
				Gateway: "6.7.8.9",
				Mac:     "second-fake-static-mac-address",
				DNS:     []string{"8.8.8.8"},
				Default: []string{"gateway", "dns"},
			}

			stubInterfaces(map[string]boshsettings.Network{
				"eth0": staticNetwork,
				"eth1": secondStaticNetwork,
			})

			interfaceAddrsProvider.GetInterfaceAddresses = []boship.InterfaceAddress{
				boship.NewSimpleInterfaceAddress("eth0", "1.2.3.4"),
				boship.NewSimpleInterfaceAddress("eth1", "5.6.7.8"),
			}

			err := netManager.SetupNetworking(boshsettings.Networks{
				"static-1": staticNetwork,
				"static-2": secondStaticNetwork,
			}, nil)
			Expect(err).ToNot(HaveOccurred())

			networkConfig0 := fs.GetFileTestStat("/etc/sysconfig/network-scripts/ifcfg-eth0")
			networkConfig1 := fs.GetFileTestStat("/etc/sysconfig/network-scripts/ifcfg-eth1")
			Expect(networkConfig0).ToNot(BeNil())
			Expect(networkConfig1).ToNot(BeNil())
			Expect(networkConfig0.StringContents()).To(Equal(`DEVICE=eth0
BOOTPROTO=static
IPADDR=1.2.3.4
NETMASK=255.255.255.0
BROADCAST=1.2.3.255
ONBOOT=yes
PEERDNS=no
DNS1=8.8.8.8
`))
			Expect(networkConfig1.StringContents()).To(Equal(`DEVICE=eth1
BOOTPROTO=static
IPADDR=5.6.7.8
NETMASK=255.255.255.0
BROADCAST=5.6.7.255
GATEWAY=6.7.8.9
ONBOOT=yes
PEERDNS=no
DNS1=8.8.8.8
`))

		})

	})

	Describe("GetConfiguredNetworkInterfaces", func() {
		Context("when there are network devices", func() {
			BeforeEach(func() {
				stubInterfaces(map[string]boshsettings.Network{
					"fake-eth0": boshsettings.Network{Mac: "aa:bb"},
					"fake-eth1": boshsettings.Network{Mac: "cc:dd"},
					"fake-eth2": boshsettings.Network{Mac: "ee:ff"},
					"fake-ens4": boshsettings.Network{Mac: "yy:zz"},
				})
			})

			writeIfcgfFile := func(iface string) {
				fs.WriteFileString(fmt.Sprintf("/etc/sysconfig/network-scripts/ifcfg-%s", iface), "fake-config")
			}

			It("returns networks that have ifcfg config present", func() {
				writeIfcgfFile("fake-eth0")
				writeIfcgfFile("fake-eth2")

				interfaces, err := netManager.GetConfiguredNetworkInterfaces()
				Expect(err).ToNot(HaveOccurred())

				Expect(interfaces).To(ConsistOf("fake-eth0", "fake-eth2"))
			})
		})

		Context("when there are no network devices", func() {
			It("returns empty list", func() {
				interfaces, err := netManager.GetConfiguredNetworkInterfaces()
				Expect(err).ToNot(HaveOccurred())
				Expect(interfaces).To(Equal([]string{}))
			})
		})
	})
}
