//go:build ignore

//Author: Charan Ravela
//Start Date: 03-20-2023
//Last Updated: 04-06-2023

#include "vmlinux.h"
#include "bpf_helpers.h"

//
struct event_data
{
    __u32 pid;
    __u32 tgid;
    __u32 uid;
    __u32 gid;
    __s32 syscall_nr;
    __u8 comm[16];
    __u8 cwd[32];
    __u8 binary_filepath[256];
    __u8 user_comm[256][256];
};
const struct event_data *unused __attribute__((unused));

struct{
    __uint(type, BPF_MAP_TYPE_RINGBUF);
    __uint(max_entries, 1 << 24);
} event SEC(".maps");

//sys_enter_execve data structure
//can be found at below path
// /sys/kernel/debug/tracing/events/syscalls/sys_enter_execve/format
struct execve_struct
{
    __u16 common_type;
    __u8 common_flags;
    __u8 common_preempt_count;
    __s32 common_pid;

    __s32 syscall_nr;
    __u8 const *filename;
    __u8 *const argv;
    __u8 *const envp;
};

SEC("tracepoint/syscalls/sys_enter_execve")
int ebpf_execve(struct execve_struct *ctx){
    struct event_data *ed;
    ed = bpf_ringbuf_reserve(&event, sizeof(struct event_data), 0);
    if (!ed) {
        return 0;
    }
    ed->syscall_nr = ctx->syscall_nr;

    //fetches user command binary filepath    
    bpf_probe_read_user(&ed->binary_filepath, sizeof(ed->binary_filepath), ctx->filename);

    // fetch current command
    bpf_get_current_comm(&ed->comm, sizeof(ed->comm));

    //fetch process id and thread group id
    __u64 pid_tgid = bpf_get_current_pid_tgid();  
    ed->pid = pid_tgid >> 32;
    ed->tgid = pid_tgid;

    //fetch user id and group id
    __u64 uid_gid = bpf_get_current_uid_gid();
    ed->uid = uid_gid >> 32;
    ed->gid = uid_gid;

    //fetches current working directory
    struct task_struct *task = (struct task_struct *)bpf_get_current_task();
    struct fs_struct *fs;
    struct dentry *dentry;

    bpf_probe_read_kernel(&fs, sizeof(fs), &task->fs);
    bpf_probe_read(&dentry, sizeof(dentry), &fs->pwd.dentry);
    bpf_probe_read(&ed->cwd, sizeof(ed->cwd), &dentry->d_iname);

    //fetches the user command
    __u8 *filn;
    int rs;
    int i = 0, j = 0;

    while (i <= 256){
        bpf_probe_read(&filn, sizeof(filn), &ctx->argv[i]);
        rs = bpf_probe_read(&ed->user_comm[j], sizeof(ed->user_comm[j]), filn);
        
        if (rs != 0){
            break;
        }

        i = i + 8;
        j++;
    };

    //pushes the information to ringbuf event map
    bpf_ringbuf_submit(ed, 0);

    return 0;
};

char _license[] SEC("license") = "Dual MIT/GPL";