#include <common/dm_log.h>
#include <linux/module.h>
#include <common/dm_kernel_funcs.h>
#include <dm_syscalls.h>

MODULE_LICENSE("GPL");
MODULE_AUTHOR("Chandler Newman");
MODULE_DESCRIPTION("Kernel based Android interception.");
MODULE_VERSION("0.1");

static int __init droidmole_init(void) {
    dm_info("DroidMole 0.1\n");
    resolve_kernel_functions();
    setup_syscall_interception();
    dm_info("Init complete\n");
    return 0;
}


static void __exit droidmole_exit(void) {
    dm_info("Unloading DroidMole!\n");
    remove_syscall_interception();
    dm_info("Unload complete\n");
}

module_init(droidmole_init);
module_exit(droidmole_exit);