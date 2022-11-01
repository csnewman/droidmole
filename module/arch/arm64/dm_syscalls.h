#ifndef DM_SYSCALLS_H
#define DM_SYSCALLS_H

#define __INTERCEPTED_SYSCALL(id, name) \
    asmlinkage long __arm64_intercepted_##name(const struct pt_regs *regs);

#include <interception/dm_syscall_table.h>

int setup_syscall_interception(void);

int remove_syscall_interception(void);

#endif //DM_SYSCALLS_H
