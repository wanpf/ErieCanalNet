
CC=clang
CFLAGS=-O2 -g  -Wall -target bpf -I/usr/include/$(shell uname -m)-linux-gnu

BPF_FS=/sys/fs/bpf
PIN_OBJECT_NS_PATH=/sys/fs/bpf
PIN_TC_GLOBAL_NS_PATH=/sys/fs/bpf/tc/globals
PROG_MOUNT_PATH=/sys/fs/bpf

MACROS:=
DEBUG ?= 1
BRIDGE_IP ?= 183763456

# see https://stackoverflow.com/questions/15063298/how-to-check-kernel-version-in-makefile
KVER = $(shell uname -r)
KMAJ = $(shell echo $(KVER) | \
sed -e 's/^\([0-9][0-9]*\)\.[0-9][0-9]*\.[0-9][0-9]*.*/\1/')
KMIN = $(shell echo $(KVER) | \
sed -e 's/^[0-9][0-9]*\.\([0-9][0-9]*\)\.[0-9][0-9]*.*/\1/')
KREV = $(shell echo $(KVER) | \
sed -e 's/^[0-9][0-9]*\.[0-9][0-9]*\.\([0-9][0-9]*\).*/\1/')

kver_ge = $(shell \
echo test | awk '{if($(KMAJ) < $(1)) {print 0} else { \
if($(KMAJ) > $(1)) {print 1} else { \
if($(KMIN) < $(2)) {print 0} else { \
if($(KMIN) > $(2)) {print 1} else { \
if($(KREV) < $(3)) {print 0} else { print 1 } \
}}}}}' \
)

# See https://nakryiko.com/posts/bpf-tips-printk/, kernel will auto print newline if version greater than 5.9.0
ifneq ($(call kver_ge,5,8,999),1)
MACROS:= $(MACROS) -DPRINTNL # kernel version less
endif

MACROS:= $(MACROS) -DBRIDGE_IP=$(BRIDGE_IP)

ifeq ($(DEBUG),1)
    MACROS:= $(MACROS) -DDEBUG
endif

CGROUP2_PATH ?= $(shell mount | grep cgroup2 | awk '{print $$3}' | grep -v "^/host" | head -n 1)
ifeq ($(CGROUP2_PATH),)
$(error It looks like your system does not have cgroupv2 enabled, or the automatic recognition fails. Please enable cgroupv2, or specify the path of cgroupv2 manually via CGROUP2_PATH parameter.)
endif

TARGETS=ecnet_cni_tc.o ecnet_cni_opts.o

$(BPF_FS):
	mountpoint -q $@ || mount -t bpf bpf $@

init-bpffs:
	mountpoint -q ${BPF_FS} || mount -t bpf bpf ${BPF_FS}
	[ -d ${PIN_TC_GLOBAL_NS_PATH} ] || sudo mkdir -p ${PIN_TC_GLOBAL_NS_PATH}

%.o: %.c
	$(CC) $(CFLAGS) $(MACROS) -c $< -o $@

generate-compilation-database:
	CC="$(CC)" CFLAGS="$(CFLAGS)" MACROS="$(MACROS)" scripts/generate-compilation-database.sh | tee compile_commands.json

compile: $(TARGETS)

compile-clean:
	[ -f ecnet_cni_tc.c ] && sudo rm -f $(TARGETS)

# Map
load-map-ecnet_dns_nat:
	[ -f $(PIN_TC_GLOBAL_NS_PATH)/ecnet_dns_nat ] || sudo bpftool map create $(PIN_TC_GLOBAL_NS_PATH)/ecnet_dns_nat type lru_hash key 12 value 8 entries 1024 name ecnet_dns_nat

load-map-ecnet_svc_nat:
	[ -f $(PIN_TC_GLOBAL_NS_PATH)/ecnet_svc_nat ] || sudo bpftool map create $(PIN_TC_GLOBAL_NS_PATH)/ecnet_svc_nat type lru_hash key 12 value 8 entries 65535 name ecnet_svc_nat

clean-maps:
	sudo rm -f \
		$(PIN_TC_GLOBAL_NS_PATH)/ecnet_dns_nat \
		$(PIN_TC_GLOBAL_NS_PATH)/ecnet_svc_nat

load-get_sockopts: load-map-ecnet_svc_nat load-map-ecnet_dns_nat
	sudo bpftool -m prog load ecnet_cni_opts.o $(PROG_MOUNT_PATH)/get_sockopts \
		map name ecnet_svc_nat pinned $(PIN_TC_GLOBAL_NS_PATH)/ecnet_svc_nat

attach-get_sockopts:
	sudo bpftool cgroup attach $(CGROUP2_PATH) getsockopt pinned $(PROG_MOUNT_PATH)/get_sockopts

clean-get_sockopts:
	sudo bpftool cgroup detach $(CGROUP2_PATH) getsockopt pinned $(PROG_MOUNT_PATH)/get_sockopts
	sudo rm $(PROG_MOUNT_PATH)/get_sockopts

load: compile init-bpffs load-get_sockopts

attach: attach-get_sockopts

clean: clean-get_sockopts clean-maps compile-clean

PROG_TC ?= /home/benne/CLionProjects/ErieCanalNet/bpf/ecnet_cni_tc.o

test-init:
	sudo ip netns add demo
	sudo ip link add veth1 type veth peer name veth0
	sudo ip link set veth0 netns demo
	sudo ip addr add 10.244.2.0/24 dev veth1
	sudo ip link set veth1 up
	sudo ip netns exec demo ip addr add 10.244.2.1/24 dev veth0
	sudo ip netns exec demo ip link set veth0 up
	sudo ip netns exec demo ip link set lo up
	sudo ip netns exec demo ip r add 0.0.0.0/0 via 10.244.2.0
	sudo iptables -t nat -A POSTROUTING -s 10.244.2.0/24 -o ens33 -j MASQUERADE
	sudo ip netns exec demo ifconfig
	sudo sed -i 's/127.0.0.53/8.8.8.8/g' /etc/resolv.conf

test-run:
	sudo ip netns exec demo tc qdisc add dev veth0 clsact
	sudo ip netns exec demo bash -c "tc filter add dev veth0 egress bpf direct-action obj $(PROG_TC) sec classifier_egress;tc filter add dev veth0 ingress bpf direct-action obj $(PROG_TC) sec classifier_ingress"

curl:
	sudo ip netns exec demo curl -I pipy-ok.pipy:8080

dns:
	sudo ip netns exec demo nslookup pipy-ok.pipy

test-clean:
	sudo ip netns exec demo tc filter delete dev veth0 egress
	sudo ip netns exec demo tc filter delete dev veth0 ingress
	sudo ip netns exec demo tc qdisc delete dev veth0 clsact

test-log:
	cat /sys/kernel/debug/tracing/trace_pipe|grep bpf_trace_printk


CRACK_NS ?= demo

crack-clean:
	sudo ip netns exec $(CRACK_NS) tc filter delete dev eth0 egress
	sudo ip netns exec $(CRACK_NS) tc filter delete dev eth0 ingress
	sudo ip netns exec $(CRACK_NS) tc qdisc delete dev eth0 clsact

crack-run:
	sudo ip netns exec $(CRACK_NS) tc qdisc add dev eth0 clsact
	sudo ip netns exec $(CRACK_NS) bash -c "tc filter add dev eth0 egress bpf direct-action obj $(PROG_TC) sec classifier_egress;tc filter add dev eth0 ingress bpf direct-action obj $(PROG_TC) sec classifier_ingress"
