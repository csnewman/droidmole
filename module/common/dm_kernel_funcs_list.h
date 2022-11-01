#include <linux/compiler_types.h>
#include <linux/kprobes.h>

#ifndef __EXPOSED_KERNEL_FUNCTION
#define __EXPOSED_KERNEL_FUNCTION(name, autores, ret, args)
#endif

// ---------------- BOOTSTRAP ----------------
__EXPOSED_KERNEL_FUNCTION(kallsyms_lookup_name, false, unsigned long, (const char *name))

// ---------------- KPROBE API ----------------
__EXPOSED_KERNEL_FUNCTION(register_kprobe, true, int, (struct kprobe *p))
__EXPOSED_KERNEL_FUNCTION(unregister_kprobe, true, void, (struct kprobe *p))

// ---------------- FILESYSTEM API ----------------
__EXPOSED_KERNEL_FUNCTION(getname, true, struct filename*, (const char __user* filename))
