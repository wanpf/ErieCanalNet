#include "headers/helpers.h"
#include "headers/maps.h"
#include <linux/bpf.h>

__section("sk_msg") int ecnet_redirmsg(struct sk_msg_md *msg)
{
    struct pair p;
    memset(&p, 0, sizeof(p));
    p.dport = bpf_htons(msg->local_port);
    p.sport = msg->remote_port >> 16;

    switch (msg->family) {
    case 2:
        // ipv4
        p.dip = msg->local_ip4;
        p.sip = msg->remote_ip4;
        break;
    }

#ifdef
    debugf("ecnet_redirmsg\tecnet_sock_pair\t\tpair.dip:dport = %pI4:%d:%d",
           &p.dip, p.dport, bpf_ntohs(p.dport));
    debugf("ecnet_redirmsg\tecnet_sock_pair\t\tpair.sip:sport = %pI4:%d:%d",
           &p.sip, p.sport, bpf_ntohs(p.sport));
#endif

    long ret = bpf_msg_redirect_hash(msg, &ecnet_sock_pair, &p, BPF_F_INGRESS);
    if (ret)
        printk("redirect %d bytes with eBPF successfully", msg->size);
    return 1;
}

char ____license[] __section("license") = "GPL";
int _version __section("version") = 1;
