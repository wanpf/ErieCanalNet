#include "headers/ecnet.h"
#include "headers/helpers.h"
#include "headers/maps.h"
#include <linux/bpf.h>

__section("cgroup/recvmsg4") int ecnet_recvmsg4(struct bpf_sock_addr *ctx)
{
    if (bpf_htons(ctx->user_port) != DNS_PROXY_PORT) {
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
    __u32 usr_ip4 = ctx->user_ip4;
    debugf("ecnet_recvmsg4 [DNS Reply]: USR IP: %pI4 PORT: %d uid: %d",
           &usr_ip4, bpf_ntohs(ctx->user_port), uid);
#endif

    __u64 cookie = bpf_get_socket_cookie_addr(ctx);
    struct origin_info *origin =
        (struct origin_info *)bpf_map_lookup_elem(&ecnet_sess_dest, &cookie);
    if (origin) {
        ctx->user_port = origin->port;
        ctx->user_ip4 = origin->ip;

#ifdef DEBUG
        debugf("ecnet_recvmsg4 successfully deal DNS redirect query");
#endif
    }
    return 1;
}

char ____license[] __section("license") = "GPL";
int _version __section("version") = 1;
