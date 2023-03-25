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

struct service_info {
    __u32 ip;
    __u16 port;
    __u16 _pad;
};

struct bpf_elf_map __section("maps") ecnet_dns_nat = {
    .type = BPF_MAP_TYPE_LRU_HASH,
    .size_key = sizeof(struct pair),
    .size_value = sizeof(struct origin_info),
    .max_elem = 1024,
    .pinning = PIN_GLOBAL_NS,
};

struct bpf_elf_map __section("maps") ecnet_svc_nat = {
    .type = BPF_MAP_TYPE_LRU_HASH,
    .size_key = sizeof(struct pair),
    .size_value = sizeof(struct origin_info),
    .max_elem = 65535,
    .pinning = PIN_GLOBAL_NS,
};