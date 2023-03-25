/*
Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/
#include "headers/ecnet.h"
#include "headers/helpers.h"
#include "headers/maps.h"
#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/in.h>
#include <linux/ip.h>
#include <linux/pkt_cls.h>
#include <linux/tcp.h>
#include <linux/udp.h>
#include <stddef.h>

#define IS_PSEUDO 0x10

#define IP_CSUM_OFF (ETH_HLEN + offsetof(struct iphdr, check))
#define IP_SRC_OFF (ETH_HLEN + offsetof(struct iphdr, saddr))
#define IP_DST_OFF (ETH_HLEN + offsetof(struct iphdr, daddr))

#define TCP_CSUM_OFF                                                           \
    (ETH_HLEN + sizeof(struct iphdr) + offsetof(struct tcphdr, check))
#define TCP_SPORT_OFF                                                          \
    (ETH_HLEN + sizeof(struct iphdr) + offsetof(struct tcphdr, source))
#define TCP_DPORT_OFF                                                          \
    (ETH_HLEN + sizeof(struct iphdr) + offsetof(struct tcphdr, dest))

#define UDP_CSUM_OFF                                                           \
    (ETH_HLEN + sizeof(struct iphdr) + offsetof(struct udphdr, check))
#define UDP_SPORT_OFF                                                          \
    (ETH_HLEN + sizeof(struct iphdr) + offsetof(struct udphdr, source))
#define UDP_DPORT_OFF                                                          \
    (ETH_HLEN + sizeof(struct iphdr) + offsetof(struct udphdr, dest))

static inline int process_tcp_ingress_packet(struct __sk_buff *skb,
                                             struct iphdr *iph, void *data_end)
{
    struct tcphdr *tcph = (struct tcphdr *)(iph + 1);
    if ((void *)(tcph + 1) > data_end) {
        return TC_ACT_SHOT;
    }

    __u32 bridge_ip = bpf_htonl(BRIDGE_IP);
    if (iph->saddr != bridge_ip) {
        return TC_ACT_OK;
    }

    __u16 bridge_port = bpf_htons(ECNET_PROXY_PORT);
    if (tcph->source != bridge_port) {
        return TC_ACT_OK;
    }

#ifdef DEBUG
    debugf("---------------------------------------------------------");
    debugf("ecnet_cni_tcp_tc [ingress]: SRC ip: %pI4 port: %d", &iph->saddr,
           bpf_ntohs(tcph->source));
    debugf("ecnet_cni_tcp_tc [ingress]: DST ip: %pI4 port: %d", &iph->daddr,
           bpf_ntohs(tcph->dest));
    debugf("ecnet_cni_tcp_tc [ingress]: FIN: %d ACK: %d", tcph->fin, tcph->ack);
#endif

    struct pair p;
    memset(&p, 0, sizeof(p));
    p.dip = iph->daddr;
    p.sip = iph->saddr;
    p.dport = tcph->dest;
    p.sport = tcph->source;

#ifdef DEBUG
    debugf("ecnet_cni_tcp_tc [ingress]: LOOKUP Pair sip: %pI4 sport: %d",
           &p.sip, bpf_htons(p.sport));
    debugf("ecnet_cni_tcp_tc [ingress]: LOOKUP Pair dip: %pI4 dport: %d",
           &p.dip, bpf_htons(p.dport));
#endif

    struct origin_info *origin = bpf_map_lookup_elem(&ecnet_ssvc_nat, &p);
    if (!origin) {
        // debugf("ecnet_cni_tcp_tc [ingress]: original not found");
        return TC_ACT_OK;
    }
    if (tcph->fin && tcph->ack) {
        // debugf("ecnet_cni_tcp_tc [ingress]: original deleted");
        bpf_map_delete_elem(&ecnet_ssvc_nat, &p);
    }

    __u32 tcp_csum_off = TCP_CSUM_OFF;
    //__u32 ip_csum_off = IP_CSUM_OFF;
    __u32 sport_off = TCP_SPORT_OFF;
    //__u32 saddr_off = IP_SRC_OFF;

    __u16 sport = tcph->source;
    //__u32 saddr = iph->saddr;
    __u16 origin_sport = origin->port;
    //__u32 origin_saddr = origin->ip;

    bpf_l4_csum_replace(skb, tcp_csum_off, sport, origin_sport, sizeof(sport));
    // bpf_l4_csum_replace(skb, tcp_csum_off, saddr, origin_saddr, IS_PSEUDO |
    // sizeof(saddr)); bpf_l3_csum_replace(skb, ip_csum_off, saddr,
    // origin_saddr, sizeof(saddr));
    bpf_skb_store_bytes(skb, sport_off, &origin_sport, sizeof(origin_sport), 0);
    // bpf_skb_store_bytes(skb, saddr_off, &origin_saddr, sizeof(origin_saddr),
    // 0);

#ifdef DEBUG
    // debugf("ecnet_cni_tcp_tc [ingress]: snat %pI4 -> %pI4", &saddr,
    // &origin->ip);
    debugf("ecnet_cni_tcp_tc [ingress]: SNAT %d -> %d", bpf_ntohs(sport),
           bpf_ntohs(origin_sport));
#endif
    return TC_ACT_OK;
}

static inline int process_udp_ingress_packet(struct __sk_buff *skb,
                                             struct iphdr *iph, void *data_end)
{
    struct udphdr *udph = (struct udphdr *)(iph + 1);
    if ((void *)(udph + 1) > data_end) {
        return TC_ACT_SHOT;
    }

    __u16 dns_port = bpf_htons(DNS_PROXY_PORT);
    if (udph->source != dns_port) {
        return TC_ACT_OK;
    }

#ifdef DEBUG
    debugf("---------------------------------------------------------");
    debugf("mcs_cni_udp_tc [ingress]: SRC ip: %pI4 port: %d", &iph->saddr,
           bpf_ntohs(udph->source));
    debugf("mcs_cni_udp_tc [ingress]: DST ip: %pI4 port: %d", &iph->daddr,
           bpf_ntohs(udph->dest));
#endif

    __u32 bridge_ip = bpf_htonl(BRIDGE_IP);
    if (iph->saddr != bridge_ip) {
        return TC_ACT_OK;
    }

    struct pair p;
    memset(&p, 0, sizeof(p));
    p.dip = iph->daddr;
    p.sip = iph->saddr;
    p.dport = udph->dest;
    p.sport = udph->source;

#ifdef DEBUG
    debugf("---------------------------------------------------------");
    debugf("mcs_cni_udp_tc [ingress]: SRC ip: %pI4 port: %d", &iph->saddr,
           bpf_ntohs(udph->source));
    debugf("mcs_cni_udp_tc [ingress]: DST ip: %pI4 port: %d", &iph->daddr,
           bpf_ntohs(udph->dest));
    debugf("mcs_cni_udp_tc [ingress]: LOOKUP Pair sip: %pI4 sport: %d", &p.sip,
           bpf_ntohs(p.sport));
    debugf("mcs_cni_udp_tc [ingress]: LOOKUP Pair dip: %pI4 dport: %d", &p.dip,
           bpf_ntohs(p.dport));
#endif

    struct origin_info *origin = bpf_map_lookup_elem(&ecnet_dns_nat, &p);
    if (!origin) {
        debugf("mcs_cni_udp_tc [ingress]: original not found");
        return TC_ACT_OK;
    }
#ifdef DEBUG
    debugf("mcs_cni_udp_tc [ingress]: LOOKUP Origin ip: %pI4 port: %d",
           &origin->ip, bpf_ntohs(origin->port));
#endif

    __u32 udp_csum_off = UDP_CSUM_OFF;
    __u32 udp_sport_off = UDP_SPORT_OFF;
    __u32 ip_csum_off = IP_CSUM_OFF;
    __u32 saddr_off = IP_SRC_OFF;
    __u32 saddr = iph->saddr;
    __u16 sport = udph->source;
    __u16 origin_sport = origin->port;

    bpf_l4_csum_replace(skb, udp_csum_off, sport, origin_sport, sizeof(sport));
    bpf_l4_csum_replace(skb, udp_csum_off, saddr, origin->ip,
                        IS_PSEUDO | sizeof(saddr));
    bpf_l3_csum_replace(skb, ip_csum_off, saddr, origin->ip, sizeof(saddr));
    bpf_skb_store_bytes(skb, saddr_off, &origin->ip, sizeof(origin->ip), 0);
    bpf_skb_store_bytes(skb, udp_sport_off, &origin_sport, sizeof(origin_sport),
                        0);

#ifdef DEBUG
    debugf("mcs_cni_udp_tc [ingress]: SNAT %pI4 -> %pI4", &saddr, &origin->ip);
#endif
    return TC_ACT_OK;
}

__section("classifier_ingress") int ecnet_cni_tc_ingress(struct __sk_buff *skb)
{
    void *data = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;
    struct ethhdr *eth = (struct ethhdr *)data;
    if ((void *)(eth + 1) > data_end) {
        return TC_ACT_SHOT;
    }

    struct iphdr *iph;

    switch (bpf_htons(eth->h_proto)) {
    case ETH_P_IP: {
        iph = (struct iphdr *)(eth + 1);
        if ((void *)(iph + 1) > data_end) {
            return TC_ACT_SHOT;
        }
        if (iph->protocol == IPPROTO_IPIP) {
            iph = ((void *)iph + iph->ihl * 4);
            if ((void *)(iph + 1) > data_end) {
                return TC_ACT_OK;
            }
        }
        if (iph->protocol == IPPROTO_TCP) {
            return process_tcp_ingress_packet(skb, iph, data_end);
        } else if (iph->protocol == IPPROTO_UDP) {
            return process_udp_ingress_packet(skb, iph, data_end);
        }
        return TC_ACT_OK;
    }
    default:
        return TC_ACT_OK;
    }
}

static inline int process_tcp_egress_packet(struct __sk_buff *skb,
                                            struct iphdr *iph, void *data_end)
{
    struct tcphdr *tcph = (struct tcphdr *)(iph + 1);
    if ((void *)(tcph + 1) > data_end) {
        return TC_ACT_SHOT;
    }

    __u32 bridge_ip = bpf_htonl(BRIDGE_IP);
    if (iph->daddr != bridge_ip) {
        return TC_ACT_OK;
    }

    __u16 bridge_port = bpf_htons(ECNET_PROXY_PORT);
    if (tcph->dest == bridge_port) {
        return TC_ACT_OK;
    }

#ifdef DEBUG
    debugf("---------------------------------------------------------");
    debugf("ecnet_cni_tcp_tc [egress]: SRC ip: %pI4 port: %d", &iph->saddr,
           bpf_ntohs(tcph->source));
    debugf("ecnet_cni_tcp_tc [egress]: DST ip: %pI4 port: %d", &iph->daddr,
           bpf_ntohs(tcph->dest));
    debugf("ecnet_cni_tcp_tc [egress]: FIN: %d ACK: %d", tcph->fin, tcph->ack);
#endif

    if (tcph->syn && !tcph->ack) {
        struct pair p;
        memset(&p, 0, sizeof(p));
        p.dip = iph->saddr;
        p.sip = bridge_ip;
        p.dport = tcph->source;
        p.sport = bridge_port;

        struct origin_info origin;
        memset(&origin, 0, sizeof(origin));
        origin.ip = iph->daddr;
        origin.port = tcph->dest;

#ifdef DEBUG
        debugf("ecnet_cni_tcp_tc [egress]: STORE Pair sip: %pI4 sport: %d",
               &p.sip, bpf_ntohs(p.sport));
        debugf("ecnet_cni_tcp_tc [egress]: STORE Pair dip: %pI4 dport: %d",
               &p.dip, bpf_ntohs(p.dport));
#endif

        bpf_map_update_elem(&ecnet_ssvc_nat, &p, &origin, BPF_NOEXIST);
    }

    __u32 tcp_csum_off = TCP_CSUM_OFF;
    //__u32 ip_csum_off = IP_CSUM_OFF;
    __u32 dport_off = TCP_DPORT_OFF;
    //__u32 daddr_off = IP_DST_OFF;

    __u16 dport = tcph->dest;
    //__u32 daddr = iph->daddr;

    bpf_l4_csum_replace(skb, tcp_csum_off, dport, bridge_port, sizeof(dport));
    // bpf_l4_csum_replace(skb, tcp_csum_off, daddr, bridge_ip, IS_PSEUDO |
    // sizeof(daddr)); bpf_l3_csum_replace(skb, ip_csum_off, daddr, bridge_ip,
    // sizeof(daddr));
    bpf_skb_store_bytes(skb, dport_off, &bridge_port, sizeof(bridge_port), 0);
    // bpf_skb_store_bytes(skb, daddr_off, &bridge_ip, sizeof(bridge_ip), 0);

#ifdef DEBUG
    // debugf("ecnet_cni_tcp_tc [egress]: dnat %pI4 -> %pI4", &daddr,
    // &bridge_ip);
    debugf("ecnet_cni_tcp_tc [egress]: DNAT %d -> %d", bpf_ntohs(dport),
           bpf_ntohs(bridge_port));
#endif
    return TC_ACT_OK;
}

static inline int process_udp_egress_packet(struct __sk_buff *skb,
                                            struct iphdr *iph, void *data_end)
{
    struct udphdr *udph = (struct udphdr *)(iph + 1);
    if ((void *)(udph + 1) > data_end) {
        return TC_ACT_SHOT;
    }

    __u16 dns_port = bpf_htons(DNS_CAPTURE_PORT);
    if (udph->dest != dns_port) {
        return TC_ACT_OK;
    }

    __u32 bridge_ip = bpf_htonl(BRIDGE_IP);
    if (iph->daddr == bridge_ip && udph->dest == dns_port) {
        return TC_ACT_OK;
    }

    __u16 bridge_port = bpf_htons(DNS_PROXY_PORT);

    struct pair p;
    memset(&p, 0, sizeof(p));
    p.dip = iph->saddr;
    p.sip = bridge_ip;
    p.dport = udph->source;
    p.sport = bridge_port;

#ifdef DEBUG
    debugf("---------------------------------------------------------");
    debugf("mcs_cni_udp_tc [egress]: SRC ip: %pI4 port: %d", &iph->saddr,
           bpf_ntohs(udph->source));
    debugf("mcs_cni_udp_tc [egress]: DST ip: %pI4 port: %d", &iph->daddr,
           bpf_ntohs(udph->dest));
    debugf("mcs_cni_udp_tc [egress]: STORE Pair sip: %pI4 sport: %d", &p.sip,
           bpf_ntohs(p.sport));
    debugf("mcs_cni_udp_tc [egress]: STORE Pair dip: %pI4 dport: %d", &p.dip,
           bpf_ntohs(p.dport));
#endif

    struct origin_info origin;
    memset(&origin, 0, sizeof(origin));
    origin.ip = iph->daddr;
    origin.port = udph->dest;

#ifdef DEBUG
    debugf("mcs_cni_udp_tc [egress]: STORE Origin ip: %pI4 port: %d",
           &origin.ip, bpf_ntohs(origin.port));
#endif
    bpf_map_update_elem(&ecnet_dns_nat, &p, &origin, BPF_NOEXIST);

    __u32 udp_csum_off = UDP_CSUM_OFF;
    __u32 udp_dport_off = UDP_DPORT_OFF;
    __u32 ip_csum_off = IP_CSUM_OFF;
    __u32 daddr_off = IP_DST_OFF;

    __u32 daddr = iph->daddr;
    __u16 dport = udph->dest;

    bpf_l4_csum_replace(skb, udp_csum_off, dport, bridge_port, sizeof(dport));
    bpf_l4_csum_replace(skb, udp_csum_off, daddr, bridge_ip,
                        IS_PSEUDO | sizeof(daddr));
    bpf_l3_csum_replace(skb, ip_csum_off, daddr, bridge_ip, sizeof(daddr));
    bpf_skb_store_bytes(skb, daddr_off, &bridge_ip, sizeof(bridge_ip), 0);
    bpf_skb_store_bytes(skb, udp_dport_off, &bridge_port, sizeof(bridge_port),
                        0);

#ifdef DEBUG
    debugf("mcs_cni_udp_tc [egress]: DNAT %pI4 -> %pI4", &origin.ip,
           &bridge_ip);
#endif
    return TC_ACT_OK;
}

__section("classifier_egress") int ecnet_cni_tc_egress(struct __sk_buff *skb)
{
    void *data = (void *)(long)skb->data;
    void *data_end = (void *)(long)skb->data_end;
    struct ethhdr *eth = (struct ethhdr *)data;
    if ((void *)(eth + 1) > data_end) {
        return TC_ACT_SHOT;
    }

    struct iphdr *iph;

    switch (bpf_htons(eth->h_proto)) {
    case ETH_P_IP: {
        iph = (struct iphdr *)(eth + 1);
        if ((void *)(iph + 1) > data_end) {
            return TC_ACT_SHOT;
        }
        if (iph->protocol == IPPROTO_IPIP) {
            iph = ((void *)iph + iph->ihl * 4);
            if ((void *)(iph + 1) > data_end) {
                return TC_ACT_OK;
            }
        }
        if (iph->protocol == IPPROTO_TCP) {
            return process_tcp_egress_packet(skb, iph, data_end);
        } else if (iph->protocol == IPPROTO_UDP) {
            return process_udp_egress_packet(skb, iph, data_end);
        }
        return TC_ACT_OK;
    }
    default:
        return TC_ACT_OK;
    }
}

char ____license[] __section("license") = "GPL";
int _version __section("version") = 1;
