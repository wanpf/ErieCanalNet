#include "headers/ecnet.h"
#include "headers/helpers.h"
#include "headers/maps.h"
#include <linux/bpf.h>

__section("cgroup/sendmsg4") int ecnet_sendmsg4(struct bpf_sock_addr *ctx)
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

#ifdef DEBUG
    debugf("ecnet_sendmsg4 [DNS Query]: dst ip: %pI4 port: %d uid: %d", &dst_ip,
           bpf_ntohs(ctx->user_port), uid);
#endif

    __u64 cookie = bpf_get_socket_cookie_addr(ctx);
    struct origin_info origin;
    memset(&origin, 0, sizeof(origin));
    origin.ip = ctx->user_ip4;
    origin.port = ctx->user_port;
    if (bpf_map_update_elem(&ecnet_sess_dest, &cookie, &origin, BPF_ANY)) {
        printk("ecnet_sendmsg4 update origin cookie failed: %d", cookie);
    }

    __u32 bridge_ip = bpf_htonl(BRIDGE_IP);
    __u16 bridge_port = bpf_htons(DNS_PROXY_PORT);

#ifdef DEBUG
    debugf("ecnet_sendmsg4 [DNS Query]: bri ip: %pI4 port: %d uid: %d",
           &bridge_ip, bpf_ntohs(bridge_port), uid);
#endif

    ctx->user_port = bridge_port;
    ctx->user_ip4 = bridge_ip;
    return 1;
}

char ____license[] __section("license") = "GPL";
int _version __section("version") = 1;
