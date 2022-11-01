#include <common/dm_log.h>
#include <dm_syscall_interception.h>
#include <common/dm_kernel_funcs.h>

DM_SYSCALL_IMPLEMENTATION3(mkdirat, int, dfd, const char __user *, pathname, umode_t, mode)
{
    dm_info("mkdirat called\n");

    struct filename* name = dmk_getname(pathname);
    dm_info("name %s \n", name->name);

//return do_mkdirat(dfd, pathname, mode);
    return 0;
}
