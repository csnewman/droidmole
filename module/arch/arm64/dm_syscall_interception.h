#ifndef DM_SYSCALL_INTERCEPTION_H
#define DM_SYSCALL_INTERCEPTION_H

#include <linux/syscalls.h>
#include <asm/syscall.h>
#include <dm_syscalls.h>

#define DM_SYSCALL_IMPLEMENTATION1(name, ...) __DM_SYSCALL_IMPLEMENTATIONx(1, name, __VA_ARGS__)
#define DM_SYSCALL_IMPLEMENTATION2(name, ...) __DM_SYSCALL_IMPLEMENTATIONx(2, name, __VA_ARGS__)
#define DM_SYSCALL_IMPLEMENTATION3(name, ...) __DM_SYSCALL_IMPLEMENTATIONx(3, name, __VA_ARGS__)
#define DM_SYSCALL_IMPLEMENTATION4(name, ...) __DM_SYSCALL_IMPLEMENTATIONx(4, name, __VA_ARGS__)
#define DM_SYSCALL_IMPLEMENTATION5(name, ...) __DM_SYSCALL_IMPLEMENTATIONx(5, name, __VA_ARGS__)
#define DM_SYSCALL_IMPLEMENTATION6(name, ...) __DM_SYSCALL_IMPLEMENTATIONx(6, name, __VA_ARGS__)

#define __DM_SYSCALL_IMPLEMENTATIONx(x, name, ...)					      	    \
	static long __se_intercepted_##name(__MAP(x,__SC_LONG,__VA_ARGS__));        \
	static inline __nocfi long __do_intercepted_##name(__MAP(x,__SC_DECL,__VA_ARGS__));	\
	asmlinkage __nocfi long __arm64_intercepted_##name(const struct pt_regs *regs)      \
	{                                                                           \
		return __se_intercepted_##name(SC_ARM64_REGS_TO_ARGS(x,__VA_ARGS__));   \
	}                                                                           \
	static __nocfi long __se_intercepted_##name(__MAP(x,__SC_LONG,__VA_ARGS__))         \
	{                                                                           \
		long ret = __do_intercepted_##name(__MAP(x,__SC_CAST,__VA_ARGS__));     \
		__MAP(x,__SC_TEST,__VA_ARGS__);                                         \
		__PROTECT(x, ret,__MAP(x,__SC_ARGS,__VA_ARGS__));                       \
		return ret;                                                             \
	}                                                                           \
	static inline __nocfi long __do_intercepted_##name(__MAP(x,__SC_DECL,__VA_ARGS__))

#endif //DM_SYSCALL_INTERCEPTION_H
