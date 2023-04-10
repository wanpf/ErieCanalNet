#include "headers/helpers.h"
#include "headers/maps.h"
#include <linux/bpf.h>
#include <linux/in.h>

#define MAX_OPS_BUFF_LENGTH 4096
#define SO_ORIGINAL_DST 80

__section("cgroup/getsockopt") int ecnet_sockopt4(struct bpf_sockopt *ctx)
{
    // currently, eBPF can not deal with optlen more than 4096 bytes, so, we
    // should limit this.
    if (ctx->optlen > MAX_OPS_BUFF_LENGTH) {
        ctx->optlen = MAX_OPS_BUFF_LENGTH;
    }
    // node proxy will call getsockopt with SO_ORIGINAL_DST, we should rewrite
    // it to return original dst info.
    if (ctx->optname != SO_ORIGINAL_DST) {
        return 1;
    }
    struct pair p;
    memset(&p, 0, sizeof(p));
    p.dport = bpf_htons(ctx->sk->src_port);
    p.sport = ctx->sk->dst_port;
    struct origin_info *origin;
    switch (ctx->sk->family) {
    case 2: // ipv4
        p.dip = ctx->sk->src_ip4;
        p.sip = ctx->sk->dst_ip4;
        origin = bpf_map_lookup_elem(&ecnet_pair_dest, &p);
#ifdef DEBUG
        debugf("ecnet_sockopt4 ecnet_pair_dest get pair.dip:dport = %pI4:%d:%d",
               &p.dip, p.dport, bpf_ntohs(p.dport));
        debugf("ecnet_sockopt4 ecnet_pair_dest get pair.sip:sport = %pI4:%d:%d",
               &p.sip, p.sport, bpf_ntohs(p.sport));
#endif
        if (origin) {
            // rewrite original_dst
            ctx->optlen = (__s32)sizeof(struct sockaddr_in);
            if ((void *)((struct sockaddr_in *)ctx->optval + 1) >
                ctx->optval_end) {
                printk("ecnet_sockopt4 optname: %d: invalid getsockopt optval",
                       ctx->optname);
                return 1;
            }
            ctx->retval = 0;
            struct sockaddr_in sa = {
                .sin_family = ctx->sk->family,
                .sin_addr.s_addr = origin->ip,
                .sin_port = origin->port,
            };
            *(struct sockaddr_in *)ctx->optval = sa;
        }
        break;
    }
    return 1;
}

char ____license[] __section("license") = "GPL";
int _version __section("version") = 1;
