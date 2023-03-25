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
#include <linux/in.h>

#define MAX_OPS_BUFF_LENGTH 4096
#define SO_ORIGINAL_DST 80

__section("cgroup/getsockopt") int get_sockopt(struct bpf_sockopt *ctx)
{
    // currently, eBPF can not deal with optlen more than 4096 bytes, so,
    // weshould limit this.
    if (ctx->optlen > MAX_OPS_BUFF_LENGTH) {
        // debugf("optname: %d, force set optlen to %d, original optlen %d is
        // too high", ctx->optname, MAX_OPS_BUFF_LENGTH, ctx->optlen);
        ctx->optlen = MAX_OPS_BUFF_LENGTH;
    }
    // App will call getsockopt with SO_ORIGINAL_DST, we should rewrite it to
    // return original dst info.
    if (ctx->optname != SO_ORIGINAL_DST) {
        return 1;
    }

    __u16 bridge_port = bpf_htons(ECNET_PROXY_PORT);
    struct pair p;
    memset(&p, 0, sizeof(p));
    struct origin_info *origin;
    switch (ctx->sk->family) {
    case 2: // ipv4
        p.sip = ctx->sk->src_ip4;
        p.dip = ctx->sk->dst_ip4;
        p.sport = bridge_port;
        p.dport = ctx->sk->dst_port;

#ifdef DEBUG
        debugf("ecnet_cni_skopts [sockopt]: LOOKUP Pair sip: %pI4 sport: %d",
               &p.sip, bpf_ntohs(p.sport));
        debugf("ecnet_cni_skopts [sockopt]: LOOKUP Pair dip: %pI4 dport: %d",
               &p.dip, bpf_ntohs(p.dport));
#endif

        origin = bpf_map_lookup_elem(&ecnet_dns_nat, &p);
        if (origin) {
#ifdef DEBUG
            debugf(
                "ecnet_cni_skopts [sockopt]: LOOKUP Origin ip: %pI4 port: %d",
                &origin->ip, bpf_ntohs(origin->port));
#endif
            // rewrite original_dst
            ctx->optlen = (__s32)sizeof(struct sockaddr_in);
            if ((void *)((struct sockaddr_in *)ctx->optval + 1) >
                ctx->optval_end) {
                printk("ecnet_cni_skopts [sockopt]: optname: %d: invalid "
                       "getsockopt optval",
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
