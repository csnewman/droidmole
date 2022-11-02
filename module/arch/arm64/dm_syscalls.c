#include <dm_syscalls.h>
#include <linux/kprobes.h>
#include <common/dm_log.h>
#include <common/dm_kernel_funcs.h>
#include <common/dm_utils.h>

static syscall_fn_t* original_sys_call_table_ptr;
static syscall_fn_t custom_sys_call_table[__NR_syscalls];

#define __STORE_ARG(t, a) regs.regs[arg++] = (__force u64) a

#define __DM_INTERCEPTED_SYSCALLx(x, id, name, ...)                   \
    __nocfi long dm_original_##name(__MAP(x,__SC_DECL,__VA_ARGS__)) { \
        struct pt_regs regs;                                          \
        int arg = 0;                                                  \
        __MAP_BLOCK(x,__STORE_ARG, __VA_ARGS__)                       \
        return original_sys_call_table_ptr[id](&regs);                \
    }

#include <interception/dm_syscall_table.h>

static int svc_handler_pre(struct kprobe *p, struct pt_regs *regs) {
    if (regs->regs[1] == __NR_mkdirat) {
        dm_info("<%s> pre_handler: p->addr = 0x%p, pc = 0x%lx,"
                " pstate = 0x%lx  0=0x%llx 1=0x%llx 2=0x%llx  3=0x%llx  exp=0x%lx    \n",
                p->symbol_name, p->addr, (long)regs->pc, (long)regs->pstate,
                regs->regs[0], regs->regs[1], regs->regs[2], regs->regs[3],
                original_sys_call_table_ptr
        );
    }

    if (regs->regs[2] == original_sys_call_table_ptr) {
        regs->regs[2] = (u64) custom_sys_call_table;
    }

    return 0;
}

static struct kprobe svc_kp = {
        .symbol_name = "el0_svc_common",
        .pre_handler = svc_handler_pre,
};

__nocfi int setup_syscall_interception(void) {
    dm_info("Resolving syscall table\n");
    original_sys_call_table_ptr = (syscall_fn_t*) dmk_kallsyms_lookup_name("sys_call_table");
    dm_info("- Syscall Table: 0x%px\n", original_sys_call_table_ptr);
    dm_info("- Syscall Table Size: 0x%px\n", sizeof (custom_sys_call_table));

    dm_info("Copying syscall table\n");
    memcpy((void*) custom_sys_call_table, (void*) original_sys_call_table_ptr, sizeof (custom_sys_call_table));

    dm_info("Patching syscall table\n");

#define __DM_INTERCEPTED_SYSCALLx(x, id, name, ...) \
    custom_sys_call_table[id] = __arm64_intercepted_##name;

#include <interception/dm_syscall_table.h>

    dm_info("Enabling syscall interception\n");
    int ret = dmk_register_kprobe(&svc_kp);
    if (ret < 0) {
        dm_info("register_kprobe failed, returned %d\n", ret);
        return ret;
    }
    dm_info("Planted kprobe at %p\n", svc_kp.addr);

    return 0;
}

__nocfi int remove_syscall_interception(void) {
    dm_info("Removing syscall interception\n");

    dmk_unregister_kprobe(&svc_kp);
    dm_info("kprobe at %p unregistered\n", svc_kp.addr);

    return 0;
}