#include "headers/ecnet.h"
#include "headers/helpers.h"
#include "headers/maps.h"
#include <linux/bpf.h>

static inline int ecnet_established(struct bpf_sock_ops *skops)
{
    __u32 bridge_ip = bpf_htonl(BRIDGE_IP);
    if (skops->local_ip4 != bridge_ip && skops->remote_ip4 != bridge_ip) {
        return 1;
    }

    __u64 cookie = bpf_get_socket_cookie_ops(skops);

    struct pair p;
    memset(&p, 0, sizeof(p));
    p.sip = skops->local_ip4;
    p.sport = bpf_htons(skops->local_port);
    p.dip = skops->remote_ip4;
    p.dport = skops->remote_port >> 16;

#ifdef DEBUG
    debugf("ecnet_established: SRC IP: %pI4 PORT: %d", &p.sip,
           bpf_ntohs(p.sport));
    debugf("ecnet_established: DST IP: %pI4 PORT: %d", &p.dip,
           bpf_ntohs(p.dport));
    debugf("ecnet_established: cookie: %d", cookie);
#endif

    struct origin_info *dst = bpf_map_lookup_elem(&ecnet_sess_dst, &cookie);
#ifdef DEBUG
    debugf("ecnet_established\tget ecnet_sess_dst\tkey:cookie = %d", cookie);
#endif
    if (dst) {
        struct origin_info dd = *dst;
        bpf_map_update_elem(&ecnet_pair_dst, &p, &dd, BPF_ANY);
#ifdef DEBUG
        debugf("ecnet_established\tset ecnet_pair_dst\tkey:pair.dip:dport = "
               "%pI4:%d:%d",
               &p.dip, p.dport, bpf_ntohs(p.dport));
        debugf("ecnet_established\tset ecnet_pair_dst\tkey:pair.sip:sport = "
               "%pI4:%d:%d",
               &p.sip, p.sport, bpf_ntohs(p.sport));
        debugf("ecnet_established\tset ecnet_pair_dst\tval:origin.ip:port = "
               "%pI4:%d:%d",
               &dd.ip, dd.port, bpf_ntohs(dd.port));
#endif
    }
    bpf_sock_hash_update(skops, &ecnet_sock_pair, &p, BPF_NOEXIST);
#ifdef DEBUG
    debugf("ecnet_sockops_ipv4\tset ecnet_sock_pair\tkey:pair.dip:dport = "
           "%pI4:%d:%d",
           &p.dip, p.dport, bpf_ntohs(p.dport));
    debugf("ecnet_sockops_ipv4\tset ecnet_sock_pair\tkey:pair.sip:sport = "
           "%pI4:%d:%d",
           &p.sip, p.sport, bpf_ntohs(p.sport));
    debugf("ecnet_sockops_ipv4\tset ecnet_sock_pair\tval:cookie = %d", cookie);
#endif
    return 0;
}

__section("sockops") int ecnet_sockops(struct bpf_sock_ops *skops)
{
    switch (skops->op) {
    case BPF_SOCK_OPS_PASSIVE_ESTABLISHED_CB:
    case BPF_SOCK_OPS_ACTIVE_ESTABLISHED_CB:
        switch (skops->family) {
        case 2:
            // AF_INET, we don't include socket.h, because it may
            // cause an import error.
            return ecnet_established(skops);
        }
        return 0;
    }
    return 0;
}

char ____license[] __section("license") = "GPL";
int _version __section("version") = 1;
