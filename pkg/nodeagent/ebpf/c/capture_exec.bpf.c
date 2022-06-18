//+build ignore
#include "vmlinux.h"
#include <bpf/bpf_helpers.h>  

#ifdef asm_inline
#undef asm_inline
#define asm_inline asm
#endif

#define ARGLEN  32
#define ARGSIZE 1024

char __license[] SEC("license") = "Dual MIT/GPL";

struct event {
	u32 pid;
	u8 comm[80];
	u8 filename[ARGSIZE];
};

// /sys/kernel/debug/tracing/events/syscalls/sys_enter_execve/format
struct trace_event_execve {
	u16 common_type;            // offset:0; size:2; signed:0;
	u8  common_flags;           // offset:2; size:1; signed:0;
	u8  common_preempt_count;   // offset:3; size:1; signed:0;
	s32 common_pid;             // offset:4; size:4; signed:1;

	s32             syscall_nr; // offset:8; size:4; signed:1;
	u32             pad;        // offset:12; size:4; signed:0; (pad)
	const u8        *filename;  // offset:16; size:8; signed:0; (ptr)
	const u8 *const *argv;      // offset:24; size:8; signed:0; (ptr)
	const u8 *const *envp;      // offset:32; size:8; signed:0; (ptr)
};

struct {
	__uint(type, BPF_MAP_TYPE_RINGBUF);
	__uint(max_entries, 1 << 24);
} events SEC(".maps");

// Zero values of any char[ARGSIZE] or char[ARGLEN][ARGSIZE] arrays.
static char zero[ARGSIZE] SEC(".rodata") = {0};
static char zero_argv[ARGLEN][ARGSIZE] SEC(".rodata") = {0};

// Force emitting struct event into the ELF.
const struct event *unused __attribute__((unused));

SEC("tracepoint/syscalls/sys_enter_execve")
s32 enter_execve(struct trace_event_execve *trace_evt) {
	u64 id   = bpf_get_current_pid_tgid();
	u32 tgid = id >> 32;
	struct event *evt;

	evt = bpf_ringbuf_reserve(&events, sizeof(struct event), 0);
	if (!evt) {
		return 0;
	}

	s64 ret = bpf_probe_read_kernel(&evt->filename, sizeof(zero), &zero);
	if (ret) {
		bpf_printk("zero out filename: %d", ret);
		bpf_ringbuf_discard(evt, 0);
		return 1;
	}

	evt->pid = tgid;
	bpf_get_current_comm(&evt->comm, 80);
	ret = bpf_probe_read_user_str(evt->filename, sizeof(evt->filename), trace_evt->filename);
	if (ret < 0) {
		bpf_printk("could not read filename into event struct: %d", ret);
		bpf_ringbuf_discard(evt, 0);
		return 1;
	}

	bpf_ringbuf_submit(evt, 0);

	return 0;
}
