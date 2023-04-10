#pragma once

#include "helpers.h"

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

struct bpf_elf_map __section("maps") ecnet_sess_dst = {
    .type = BPF_MAP_TYPE_LRU_HASH,
    .size_key = sizeof(__u64),
    .size_value = sizeof(struct origin_info),
    .max_elem = 65535,
};

struct bpf_elf_map __section("maps") ecnet_pair_dst = {
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