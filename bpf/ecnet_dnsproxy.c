#include "headers/ecnet.h"
#include "headers/helpers.h"
#include "headers/maps.h"
#include <linux/bpf.h>
#include <linux/if_ether.h>
#include <linux/in.h>
#include <linux/ip.h>
#include <linux/udp.h>
#include <linux/version.h>
#include <string.h>

char dns_buffer[512];

static void swap_src_dst_mac(struct ethhdr *eth);
static void swap_src_dst_udp(struct udphdr *udph);
static __u16 cal_udp_csum(struct iphdr *iph, struct udphdr *udph,
                          void *data_end);
static __u16 csum_fold_helper(__u64 csum);
static void ipv4_csum(void *data_start, int data_size, __u64 *csum);
static int match_a_records(struct xdp_md *ctx, struct dns_query *q,
                           struct a_record *a);
static int parse_query(struct xdp_md *ctx, void *query_start,
                       struct dns_query *q);
static void create_query_response(struct a_record *a, char *dns_buffer,
                                  size_t *buf_size);
static inline void modify_dns_header_response(struct dnshdr *dns_hdr);
static inline void copy_to_pkt_buf(struct xdp_md *ctx, void *dst, void *src,
                                   size_t n);

#define MAX_UDP_SIZE 1480

#ifndef memcpy
#define memcpy(dest, src, n) __builtin_memcpy((dest), (src), (n))
#endif

__section("prog") int ecnet_dns_proxy(struct xdp_md *ctx)
{
#ifdef DEBUG
    __u64 start = bpf_ktime_get_ns();
#endif

    void *data_end = (void *)(unsigned long)ctx->data_end;
    void *data = (void *)(unsigned long)ctx->data;

    // Boundary check: check if packet is larger than a full ethernet + ip
    // header
    if (data + sizeof(struct ethhdr) + sizeof(struct iphdr) > data_end) {
        return XDP_PASS;
    }

    struct ethhdr *eth = data;

    // Ignore packet if ethernet protocol is not IP-based
    if (eth->h_proto != bpf_htons(ETH_P_IP)) {
        return XDP_PASS;
    }

    struct iphdr *ip = data + sizeof(*eth);

    if (ip->protocol != IPPROTO_UDP) {
        return XDP_PASS;
    }

    // Boundary check for minimal DNS header
    if (data + sizeof(struct ethhdr) + sizeof(struct iphdr) +
            sizeof(struct udphdr) + sizeof(struct dnshdr) >
        data_end) {
        return XDP_PASS;
    }

    __u32 eip = ip->daddr;
    __u32 *cluster_ip = bpf_map_lookup_elem(&ecnet_dns_eips, &eip);
    // If record pointer is zero
    if (!cluster_ip) {
        return XDP_PASS;
    }
    struct udphdr *udp = data + sizeof(*eth) + sizeof(*ip);

    // Check if dest port equals 53
    if (udp->dest == bpf_htons(53)) {
#ifdef DEBUG
        debugf("-------------------------------");
        debugf("udp dst ip:%pI4 port:%d", &ip->daddr, bpf_ntohs(udp->dest));
        debugf("udp src ip:%pI4 port:%d", &ip->saddr, bpf_ntohs(udp->source));
        debugf("Packet dest port 53");
        debugf("Data pointer starts at %u", data);
#endif

        struct dnshdr *dns_hdr =
            data + sizeof(*eth) + sizeof(*ip) + sizeof(*udp);
        // Check if header contains a standard query
        if (dns_hdr->qr == 0 && dns_hdr->opcode == 0) {
#ifdef DEBUG
            debugf("DNS query transaction id %u",
                   bpf_ntohs(dns_hdr->transaction_id));
#endif

            // Get a pointer to the start of the DNS query
            void *query_start = (void *)dns_hdr + sizeof(struct dnshdr);

            // We will only be parsing a single query for now
            struct dns_query q;
            int query_length = 0;
            query_length = parse_query(ctx, query_start, &q);
            if (query_length < 1) {
                return XDP_PASS;
            }

            // Check if query matches a record in our hash table
            struct a_record a_record;
            int res = match_a_records(ctx, &q, &a_record);
            // If query matches...
            if (res == 0) {
                size_t buf_size = 0;

                // Change DNS header to a valid response header
                modify_dns_header_response(dns_hdr);

                // Create DNS response and add to temporary buffer.
                create_query_response(&a_record, &dns_buffer[buf_size],
                                      &buf_size);

                // Start our response [query_length] bytes beyond the header
                void *answer_start =
                    (void *)dns_hdr + sizeof(struct dnshdr) + query_length;
                // Determine increment of packet buffer
                int adjust = answer_start + buf_size - data_end;

                // Adjust packet length accordingly
                if (bpf_xdp_adjust_tail(ctx, adjust)) {
#ifdef DEBUG
                    debugf("Adjust tail fail");
#endif
                } else {
                    // Because we adjusted packet length, mem addresses might be
                    // changed. Reinit pointers, as verifier will complain
                    // otherwise.
                    data = (void *)(unsigned long)ctx->data;
                    data_end = (void *)(unsigned long)ctx->data_end;

                    // Copy bytes from our temporary buffer to packet buffer
                    copy_to_pkt_buf(ctx,
                                    data + sizeof(struct ethhdr) +
                                        sizeof(struct iphdr) +
                                        sizeof(struct udphdr) +
                                        sizeof(struct dnshdr) + query_length,
                                    &dns_buffer[0], buf_size);

                    eth = data;
                    ip = data + sizeof(struct ethhdr);
                    udp = data + sizeof(struct ethhdr) + sizeof(struct iphdr);

                    // Do a new boundary check
                    if (data + sizeof(struct ethhdr) + sizeof(struct iphdr) +
                            sizeof(struct udphdr) >
                        data_end) {
#ifdef DEBUG
                        debugf("Error: Boundary exceeded");
#endif
                        return XDP_PASS;
                    }

                    // Adjust UDP length and IP length
                    __u16 iplen = (data_end - data) - sizeof(struct ethhdr);
                    __u16 udplen = (data_end - data) - sizeof(struct ethhdr) -
                                   sizeof(struct iphdr);
                    ip->tot_len = bpf_htons(iplen);
                    udp->len = bpf_htons(udplen);

                    // Swap src/dst IP
                    __u32 src_ip = ip->saddr;
                    ip->saddr = *cluster_ip;
                    ip->daddr = src_ip;

                    /* Swap UDP source and destination */
                    swap_src_dst_udp(udp);
                    /* Swap IP source and destination */
                    // swap_src_dst_ipv4(ip);
                    /* Swap Ethernet source and destination */
                    swap_src_dst_mac(eth);

                    /* Update IP checksum */
                    __u64 csum = 0;
                    ip->check = 0;
                    ipv4_csum(ip, sizeof(struct iphdr), &csum);
                    ip->check = csum;

                    /* Update UDP checksum */
                    udp->check = 0;
                    udp->check = cal_udp_csum(ip, udp, data_end);

#ifdef DEBUG
                    debugf("XDP_TX");
                    __u64 end = bpf_ktime_get_ns();
                    __u64 elapsed = end - start;
                    debugf("Time elapsed: %d", elapsed);
#endif

                    // Emit modified packet
                    return XDP_TX;
                }
            }
        }
    }

    return XDP_PASS;
}

static __always_inline void swap_src_dst_mac(struct ethhdr *eth)
{
    unsigned char tmp[ETH_ALEN];
    memcpy(tmp, eth->h_source, ETH_ALEN);
    memcpy(eth->h_source, eth->h_dest, ETH_ALEN);
    memcpy(eth->h_dest, tmp, ETH_ALEN);
    return;
}

static __always_inline void swap_src_dst_udp(struct udphdr *udph)
{
    __be16 tmp = udph->source;
    udph->source = udph->dest;
    udph->dest = tmp;
    return;
}

static __always_inline __u16 cal_udp_csum(struct iphdr *iph,
                                          struct udphdr *udph, void *data_end)
{
    __u32 csum_buffer = 0;
    __u16 *buf = (void *)udph;

    // Compute pseudo-header checksum
    csum_buffer += (__u16)iph->saddr;
    csum_buffer += (__u16)(iph->saddr >> 16);
    csum_buffer += (__u16)iph->daddr;
    csum_buffer += (__u16)(iph->daddr >> 16);
    csum_buffer += (__u16)iph->protocol << 8;
    csum_buffer += udph->len;

    // Compute checksum on udp header + payload
    for (int i = 0; i < MAX_UDP_SIZE; i += 2) {
        if ((void *)(buf + 1) > data_end) {
            break;
        }
        csum_buffer += *buf;
        buf++;
    }
    if ((void *)buf + 1 <= data_end) {
        // In case payload is not 2 bytes aligned
        csum_buffer += *(__u8 *)buf;
    }

    __u16 csum = (__u16)csum_buffer + (__u16)(csum_buffer >> 16);
    csum = ~csum;

    return csum;
}

static __always_inline __u16 csum_fold_helper(__u64 csum)
{
    int i;
#pragma unroll
    for (i = 0; i < 4; i++) {
        if (csum >> 16)
            csum = (csum & 0xffff) + (csum >> 16);
    }
    return ~csum;
}

static __always_inline void ipv4_csum(void *data_start, int data_size,
                                      __u64 *csum)
{
    *csum = bpf_csum_diff(0, 0, data_start, data_size, *csum);
    *csum = csum_fold_helper(*csum);
    return;
}

static int match_a_records(struct xdp_md *ctx, struct dns_query *q,
                           struct a_record *a)
{
#ifdef DEBUG
    debugf("DNS record type: %i", q->record_type);
    debugf("DNS class: %i", q->class);
    // debugf("DNS name: %s", q->name);
#endif

    struct a_record *record;
    record = bpf_map_lookup_elem(&ecnet_dns_aaaa, q);

    // If record pointer is not zero..
    if (record > 0) {
#ifdef DEBUG
        debugf("DNS query matched");
#endif

        a->ip_addr = record->ip_addr;
        a->ttl = record->ttl;
        return 0;
    }
    return -1;
}

// Parse query and return query length
static int parse_query(struct xdp_md *ctx, void *query_start,
                       struct dns_query *q)
{
    void *data_end = (void *)(long)ctx->data_end;

#ifdef DEBUG
    debugf("Parsing query");
#endif

    __u16 i;
    void *cursor = query_start;
    int namepos = 0;

    // Fill dns_query.name with zero bytes
    // Not doing so will make the verifier complain when dns_query is used as a
    // key in bpf_map_lookup
    memset(&q->name[0], 0, sizeof(q->name));
    // Fill record_type and class with default values to satisfy verifier
    q->record_type = 0;
    q->class = 0;

    // We create a bounded loop of MAX_DNS_NAME_LENGTH (maximum allowed dns name
    // size). We'll loop through the packet byte by byte until we reach '0' in
    // order to get the dns query name
    for (i = 0; i < MAX_DNS_NAME_LENGTH; i++) {

        // Boundary check of cursor. Verifier requires a +1 here.
        // Probably because we are advancing the pointer at the end of the loop
        if (cursor + 1 > data_end) {
#ifdef DEBUG
            debugf("Error: boundary exceeded while parsing DNS query name");
#endif
            break;
        }

        // If separator is zero we've reached the end of the domain query
        if (*(char *)(cursor) == 0) {

            // We've reached the end of the query name.
            // This will be followed by 2x 2 bytes: the dns type and dns class.
            if (cursor + 5 > data_end) {
#ifdef DEBUG
                debugf(
                    "Error: boundary exceeded while retrieving DNS record type "
                    "and class");
#endif
            } else {
                q->record_type = bpf_htons(*(__u16 *)(cursor + 1));
                q->class = bpf_htons(*(__u16 *)(cursor + 3));
            }

            // Return the bytecount of (namepos + current '0' byte + dns type +
            // dns class) as the query length.
            return namepos + 1 + 2 + 2;
        }

        // Read and fill data into struct
        q->name[namepos] = *(char *)(cursor);
        namepos++;
        cursor++;
    }

    return -1;
}

static void create_query_response(struct a_record *a, char *dns_buffer,
                                  size_t *buf_size)
{
    // Formulate a DNS response. Currently defaults to hardcoded query pointer +
    // type a + class in + ttl + 4 bytes as reply.
    struct dns_response *response = (struct dns_response *)&dns_buffer[0];
    response->query_pointer = bpf_htons(0xc00c);
    response->class = bpf_htons(DNS_CLASS_IN);
    response->ttl = bpf_htonl(a->ttl);

    response->record_type = bpf_htons(A_RECORD_TYPE);
    response->data_length = bpf_htons(sizeof(a->ip_addr));
    *buf_size += sizeof(struct dns_response);
    // Copy IP address
    __builtin_memcpy(&dns_buffer[*buf_size], &a->ip_addr,
                     sizeof(struct in_addr));
    *buf_size += sizeof(struct in_addr);
}

static inline void modify_dns_header_response(struct dnshdr *dns_hdr)
{
    // Set query response
    dns_hdr->qr = 1;
    // Set truncated to 0
    // dns_hdr->tc = 0;
    // Set authorative to zero
    // dns_hdr->aa = 0;
    // Recursion available
    dns_hdr->ra = 1;
    // One answer
    dns_hdr->ans_count = bpf_htons(1);
}

//__builtin_memcpy only supports static size_t
// The following function is a memcpy wrapper that uses __builtin_memcpy when
// size_t n is known. Otherwise it uses our own naive & slow memcpy routine
static inline void copy_to_pkt_buf(struct xdp_md *ctx, void *dst, void *src,
                                   size_t n)
{
    // Boundary check
    if ((void *)(long)ctx->data_end >= dst + n) {
        int i;
        char *cdst = dst;
        char *csrc = src;

        // For A records, src is either 16 or 27 bytes, depending if OPT record
        // is requested. Use __builtin_memcpy for this. Otherwise, use our own
        // slow, naive memcpy implementation.
        switch (n) {
        case 16:
            __builtin_memcpy(cdst, csrc, 16);
            break;

        case 27:
            __builtin_memcpy(cdst, csrc, 27);
            break;

        default:
            for (i = 0; i < n; i += 1) {
                cdst[i] = csrc[i];
            }
        }
    }
}

char _license[] __section("license") = "GPL";
__u32 _version __section("version") = LINUX_VERSION_CODE;
