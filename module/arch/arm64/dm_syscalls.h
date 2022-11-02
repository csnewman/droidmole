#ifndef DM_SYSCALLS_H
#define DM_SYSCALLS_H
#include <linux/syscalls.h>
#include <asm/syscall.h>

#define __DM_INTERCEPTED_SYSCALLx(x, id, name, ...)                         \
    asmlinkage long __arm64_intercepted_##name(const struct pt_regs *regs); \
    long dm_original_##name(__MAP(x,__SC_DECL,__VA_ARGS__));

#include <interception/dm_syscall_table.h>

int setup_syscall_interception(void);

int remove_syscall_interception(void);

#endif //DM_SYSCALLS_H
