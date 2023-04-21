#pragma once

#include "helpers.h"

#define A_RECORD_TYPE 0x0001
#define DNS_CLASS_IN 0x0001
// RFC1034: the total number of octets that represent a domain name is limited
// to 255. We need to be aligned so the struct does not include padding bytes.
// We'll set the length to 256. Otherwise padding bytes will generate problems
// with the verifier, as it ?could contain arbitrary data from memory?
#define MAX_DNS_NAME_LENGTH 256

struct dnshdr {
    __u16 transaction_id;
    __u8 rd : 1;      // Recursion desired
    __u8 tc : 1;      // Truncated
    __u8 aa : 1;      // Authoritive answer
    __u8 opcode : 4;  // Opcode
    __u8 qr : 1;      // Query/response flag
    __u8 rcode : 4;   // Response code
    __u8 cd : 1;      // Checking disabled
    __u8 ad : 1;      // Authenticated data
    __u8 z : 1;       // Z reserved bit
    __u8 ra : 1;      // Recursion available
    __u16 q_count;    // Number of questions
    __u16 ans_count;  // Number of answer RRs
    __u16 auth_count; // Number of authority RRs
    __u16 add_count;  // Number of resource RRs
};

// Used as key in our hashmap
struct dns_query {
    __u16 record_type;
    __u16 class;
    char name[MAX_DNS_NAME_LENGTH];
};

// Used as a generic DNS response
struct dns_response {
    __u16 query_pointer;
    __u16 record_type;
    __u16 class;
    __u32 ttl;
    __u16 data_length;
} __attribute__((packed));

// Used as value of our A record hashmap
struct a_record {
    struct in_addr ip_addr;
    __u32 ttl;
};

struct pair {
    __u32 sip;
    __u32 dip;
    __u16 sport;
    __u16 dport;
};

struct origin_info {
    __u32 ip;
    __u16 port;
    __u16 _pad;
};

// Hash table for DNS A Records loaded by iproute2
// Key is a dns_query struct, value is the associated IPv4 address
struct bpf_elf_map __section("maps") ecnet_dns_aaaa = {
    .type = BPF_MAP_TYPE_LRU_HASH,
    .size_key = sizeof(struct dns_query),
    .size_value = sizeof(struct a_record),
    .max_elem = 65535,
    .pinning = 2, // PIN_GLOBAL_NS
};

struct bpf_elf_map __section("maps") ecnet_dns_eips = {
    .type = BPF_MAP_TYPE_LRU_HASH,
    .size_key = sizeof(__u32),
    .size_value = sizeof(__u32),
    .max_elem = 65535,
    .pinning = 2, // PIN_GLOBAL_NS
};

struct bpf_elf_map __section("maps") ecnet_sess_dest = {
    .type = BPF_MAP_TYPE_LRU_HASH,
    .size_key = sizeof(__u64),
    .size_value = sizeof(struct origin_info),
    .max_elem = 65535,
};

struct bpf_elf_map __section("maps") ecnet_pair_dest = {
    .type = BPF_MAP_TYPE_LRU_HASH,
    .size_key = sizeof(struct pair),
    .size_value = sizeof(struct origin_info),
    .max_elem = 65535,
};

struct bpf_elf_map __section("maps") ecnet_sock_pair = {
    .type = BPF_MAP_TYPE_SOCKHASH,
    .size_key = sizeof(struct pair),
    .size_value = sizeof(__u32),
    .max_elem = 65535,
};