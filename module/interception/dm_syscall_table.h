#include <linux/syscalls.h>
#include <asm/syscall.h>

#ifndef __DM_INTERCEPTED_SYSCALLx
#define __DM_INTERCEPTED_SYSCALLx(x, id, name, ...)
#endif

#define DM_INTERCEPTED_SYSCALL1(id, name, ...) __DM_INTERCEPTED_SYSCALLx(1, id, name, __VA_ARGS__)
#define DM_INTERCEPTED_SYSCALL2(id, name, ...) __DM_INTERCEPTED_SYSCALLx(2, id, name, __VA_ARGS__)
#define DM_INTERCEPTED_SYSCALL3(id, name, ...) __DM_INTERCEPTED_SYSCALLx(3, id, name, __VA_ARGS__)
#define DM_INTERCEPTED_SYSCALL4(id, name, ...) __DM_INTERCEPTED_SYSCALLx(4, id, name, __VA_ARGS__)
#define DM_INTERCEPTED_SYSCALL5(id, name, ...) __DM_INTERCEPTED_SYSCALLx(5, id, name, __VA_ARGS__)
#define DM_INTERCEPTED_SYSCALL6(id, name, ...) __DM_INTERCEPTED_SYSCALLx(6, id, name, __VA_ARGS__)

// ---------------- FILESYSTEM ----------------
DM_INTERCEPTED_SYSCALL3(__NR_mkdirat, mkdirat, int, dfd, const char __user *, pathname, umode_t, mode)
