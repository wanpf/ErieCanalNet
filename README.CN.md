## veth test

```
ip netns add demo
ip link add veth1 type veth peer name veth0
ip link set veth0 netns demo

ip addr add 1.0.0.1/24 dev veth1
ip link set veth1 up


ip netns exec demo ip addr add 1.0.0.2/24 dev veth0
ip netns exec demo ip link set veth0 up


ip netns exec demo ip link set lo up
ip netns exec demo ip r del 0.0.0.0/0
ip netns exec demo ip r add 0.0.0.0/0 via 1.0.0.1
iptables -t nat -A POSTROUTING -s 1.0.0.0/24 -o ens33 -j MASQUERADE
ip netns exec demo ifconfig
ip netns exec demo ping 1.0.0.1 
ip netns exec demo ping 8.8.8.8

ip netns exec demo curl 1.0.0.1/hello


sudo ip netns exec ns1 python3 app.80.py

sudo ip netns exec ns1 python3 app.90.py

cat /sys/kernel/debug/tracing/trace_pipe|grep bpf_trace_printk

prog=/home/benne/CLionProjects/ecnet-edge/bpf/ecnet_cni_tc_nat.o

sudo ip netns exec demo tc qdisc add dev veth0 clsact
sudo ip netns exec demo tc filter add dev veth0 egress bpf direct-action obj ${prog} sec classifier_egress

sudo ip netns exec demo tc filter delete dev veth0 egress
sudo ip netns exec demo tc qdisc delete dev veth0 clsact

sudo ip netns exec demo tcpdump eth0 tcp

192.168.127.52 3232268084
1.0.0.1 16777217
1.0.0.2 16777218

1.0.0.3 16777219

8.8.8.8 134744072
169.254.168.252 2852038908
https://www.bejson.com/convert/ip2int/

```

## Crack code

```bash
pipy --log-level=debug -e "pipy().listen('1.0.0.1',8080).serveHTTP(new Message('Hi, I am pipy ok v1'))"
```

