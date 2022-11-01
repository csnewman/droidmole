#include <asm/syscall.h>

#ifndef __INTERCEPTED_SYSCALL
#define __INTERCEPTED_SYSCALL(id, name)
#endif

// ---------------- FILESYSTEM ----------------
__INTERCEPTED_SYSCALL(__NR_mkdirat, mkdirat)
