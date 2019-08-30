package main

import (
	"github.com/google/gopacket"
	"github.com/google/gopacket/layers"
	"log"
	"net"
	"time"
)

var IPWhiteList map[string]int64

func RXLoop(Handle uintptr) {
	for {
		c := make(chan *DivertPacket, 1)
		go func() {
			Packet, err := WinDivertRecv(Handle)
			if err != nil {
				log.Println(err)
				c <- nil
			}
			c <- Packet
		}()
		select {
		case Packet := <-c:
			if Packet != nil {
				RXChan <- Packet
			} else {
				log.Println("RXLoop Stop")
				return
			}
		case <-time.After(1 * time.Second):
			if EndFlag {
				log.Println("RXLoop Stop")
				return
			} else {
				continue
			}
		}
	}
}

func PXLoop(Handle uintptr) {
	for {
		select {
		case Packet := <-RXChan:
			IPVersion := Packet.Data[0] >> 4
			var thisPacket gopacket.Packet
			var SrcIP, DstIP net.IP
			if IPVersion == 4 {
				thisPacket = gopacket.NewPacket(Packet.Data, layers.LayerTypeIPv4, gopacket.Lazy)
				IPHeader := thisPacket.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
				SrcIP = IPHeader.SrcIP
				DstIP = IPHeader.DstIP
			} else {
				thisPacket = gopacket.NewPacket(Packet.Data, layers.LayerTypeIPv6, gopacket.Lazy)
				IPHeader := thisPacket.Layer(layers.LayerTypeIPv6).(*layers.IPv6)
				SrcIP = IPHeader.SrcIP
				DstIP = IPHeader.DstIP
			}
			if (Packet.Addr.Flag>>17)%2 == 1 {
				//Outbound Pass Directly and Record
				IPWhiteList[DstIP.String()] = time.Now().Unix()
				TXChan <- Packet
			} else {
				//Inbound
				_, Exist := IPWhiteList[SrcIP.String()]
				if Exist {
					//Exist in WhiteList
					TXChan <- Packet
				} else {
					log.Println("Maybe Detection")
					log.Println(thisPacket)
					//Drop
					buffer := gopacket.NewSerializeBuffer()
					options := gopacket.SerializeOptions{}
					options.ComputeChecksums = true
					if IPVersion == 4 {
						thisPacket = gopacket.NewPacket(Packet.Data, layers.LayerTypeIPv4, gopacket.Lazy)
						IPHeader := thisPacket.Layer(layers.LayerTypeIPv4).(*layers.IPv4)
						IPHeader.DstIP = net.ParseIP(configuration.V4RedirectAddr)
						gopacket.SerializeLayers(buffer, options, IPHeader)
					} else {
						thisPacket = gopacket.NewPacket(Packet.Data, layers.LayerTypeIPv6, gopacket.Lazy)
						IPHeader := thisPacket.Layer(layers.LayerTypeIPv6).(*layers.IPv6)
						IPHeader.DstIP = net.ParseIP(configuration.V6RedirectAddr)
						gopacket.SerializeLayers(buffer, options, IPHeader)
					}
					if SendOut(Handle, buffer.Bytes()) != nil {
						log.Println("Redirect Failed")
					}
				}
			}
		case <-time.After(1 * time.Second):
			if EndFlag {
				log.Println("PXLoop Stop")
				return
			} else {
				continue
			}
		}
	}
}

func TXLoop(Handle uintptr) {
	for {
		select {
		case Packet := <-TXChan:
			err := WinDivertSend(Handle, Packet)
			if err != nil {
				log.Println(err)
				log.Println("TXLoop Stop")
				return
			}
		case <-time.After(1 * time.Second):
			if EndFlag {
				log.Println("TXLoop Stop")
				return
			} else {
				continue
			}
		}
	}
}
