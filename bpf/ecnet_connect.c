#include "headers/ecnet.h"
#include "headers/helpers.h"
#include "headers/maps.h"
#include <linux/bpf.h>
#include <linux/in.h>

static inline int ecnet_udp_connect4(struct bpf_sock_addr *ctx)
{
    if (bpf_htons(ctx->user_port) != DNS_CAPTURE_PORT) {
        return 1;
    }

    __u64 uid = bpf_get_current_uid_gid() & 0xffffffff;
    if (uid == 0 || uid == PROXY_USER_ID) {
        return 1;
    }

    __u32 dst_ip = ctx->user_ip4;
    if ((dst_ip & 0xff) == 0x7f) {
        // call local, bypass.
        return 1;
    }

    __u64 cookie = bpf_get_socket_cookie_addr(ctx);

#ifdef DEBUG_DNS
    __u64 cgrp_id = bpf_get_current_cgroup_id();
    debugf("ecnet_udp_connect4 [DNS Query]: DST IP: %pI4 PORT: %d", &dst_ip,
           bpf_ntohs(ctx->user_port));
    debugf("ecnet_udp_connect4 [DNS Query]: CKI: %d CGID: %d UID: %d", cookie,
           cgrp_id, uid);
#endif

    struct origin_info origin;
    memset(&origin, 0, sizeof(origin));
    origin.ip = ctx->user_ip4;
    origin.port = ctx->user_port;

    if (bpf_map_update_elem(&ecnet_sess_dst, &cookie, &origin, BPF_ANY)) {
        printk(
            "ecnet_udp_connect4 [DNS Query]: Update origin cookie failed: %d",
            cookie);
    }

    __u32 bridge_ip = bpf_htonl(BRIDGE_IP);
    __u16 bridge_port = bpf_htons(DNS_PROXY_PORT);

#ifdef DEBUG_DNS
    debugf("ecnet_udp_connect4 [DNS Query]: BRI IP: %pI4 PORT: %d UID: %d",
           &bridge_ip, bpf_ntohs(bridge_port), uid);
#endif

    ctx->user_port = bridge_port;
    ctx->user_ip4 = bridge_ip;
    return 1;
}

static inline int ecnet_tcp_connect4(struct bpf_sock_addr *ctx)
{
    if (bpf_htons(ctx->user_port) == ECNET_PROXY_PORT) {
        return 1;
    }

    __u32 dst_ip = ctx->user_ip4;
    if ((dst_ip & 0xff) == 0x7f) {
        // call local, bypass.
        return 1;
    }

    __u32 bridge_ip = bpf_htonl(BRIDGE_IP);
    if (dst_ip != bridge_ip) {
        return 1;
    }

    __u64 uid = bpf_get_current_uid_gid() & 0xffffffff;
    if (uid == 0 || uid == PROXY_USER_ID) {
        return 1;
    }

#ifdef
    __u64 cookie = bpf_get_socket_cookie_addr(ctx);
    __u64 cgrp_id = bpf_get_current_cgroup_id();
    debugf("ecnet_tcp_connect4 : DST IP: %pI4 PORT: %d", &dst_ip,
           bpf_ntohs(ctx->user_port));
    debugf("ecnet_tcp_connect4 : CKI: %d CGID: %d UID: %d", cookie, cgrp_id,
           uid);
#endif

    // redirect it to node proxy.
    struct origin_info origin;
    memset(&origin, 0, sizeof(origin));
    origin.ip = dst_ip;
    origin.port = ctx->user_port;

#ifdef
    debugf("ecnet_tcp_connect4\tset ecnet_sess_dst\tkey:cookie = %d", cookie);
    debugf("ecnet_tcp_connect4\tset ecnet_sess_dst\tval:origin.ip:port = "
           "%pI4:%d:%d",
           &origin.ip, origin.port, bpf_ntohs(origin.port));
#endif

    // origin.flags = 1;
    if (bpf_map_update_elem(&ecnet_sess_dst, &cookie, &origin, BPF_ANY)) {
        printk("write ecnet_sess_dst failed");
        return 0;
    }

    ctx->user_port = bpf_htons(ECNET_PROXY_PORT);

    return 1;
}

__section("cgroup/connect4") int ecnet_sock_connect4(struct bpf_sock_addr *ctx)
{
    switch (ctx->protocol) {
    case IPPROTO_TCP:
        return ecnet_tcp_connect4(ctx);
    case IPPROTO_UDP:
        return ecnet_udp_connect4(ctx);
    default:
        return 1;
    }
}

char ____license[] __section("license") = "GPL";
int _version __section("version") = 1;
