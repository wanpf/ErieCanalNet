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

#ifdef DEBUG
    debugf("ecnet_redirmsg ecnet_sock_pair dir pair.dip:dport = %pI4:%d",
           &p.dip, bpf_ntohs(p.dport));
    debugf("ecnet_redirmsg ecnet_sock_pair dir pair.sip:sport = %pI4:%d",
           &p.sip, bpf_ntohs(p.sport));
    long ret = bpf_msg_redirect_hash(msg, &ecnet_sock_pair, &p, BPF_F_INGRESS);
    if (ret)
        printk(
            "[debug] ecnet_redirmsg redirect %d bytes with eBPF successfully",
            msg->size);
#else
    bpf_msg_redirect_hash(msg, &ecnet_sock_pair, &p, BPF_F_INGRESS);
#endif

    return 1;
}

char ____license[] __section("license") = "GPL";
int _version __section("version") = 1;
