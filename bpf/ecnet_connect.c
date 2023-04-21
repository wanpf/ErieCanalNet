#include "headers/ecnet.h"
#include "headers/helpers.h"
#include "headers/maps.h"
#include <linux/bpf.h>
#include <linux/in.h>

static inline int ecnet_tcp_con4(struct bpf_sock_addr *ctx)
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

    __u64 cookie = bpf_get_socket_cookie_addr(ctx);
#ifdef DEBUG
    debugf("ecnet_tcp_con4 dst ip: %pI4 port: %d", &dst_ip,
           bpf_ntohs(ctx->user_port));
    debugf("ecnet_tcp_con4 cke: %d uid: %d", cookie, uid);
#endif

    // redirect it to node proxy.
    struct origin_info origin;
    memset(&origin, 0, sizeof(origin));
    origin.ip = dst_ip;
    origin.port = ctx->user_port;

#ifdef DEBUG
    debugf("ecnet_tcp_con4 ecnet_sess_dest set key:cookie = %d", cookie);
    debugf("ecnet_tcp_con4 ecnet_sess_dest set val:origin.ip:port = %pI4:%d",
           &origin.ip, bpf_ntohs(origin.port));
#endif

    // origin.flags = 1;
    if (bpf_map_update_elem(&ecnet_sess_dest, &cookie, &origin, BPF_ANY)) {
        printk("ecnet_tcp_con4 write ecnet_sess_dest failed");
        return 0;
    }

    ctx->user_port = bpf_htons(ECNET_PROXY_PORT);

    return 1;
}

__section("cgroup/connect4") int ecnet_sock_connect4(struct bpf_sock_addr *ctx)
{
    switch (ctx->protocol) {
    case IPPROTO_TCP:
        return ecnet_tcp_con4(ctx);
    default:
        return 1;
    }
}

char ____license[] __section("license") = "GPL";
int _version __section("version") = 1;
