#include <linux/slab.h>
#include <common/dm_kernel_funcs.h>
#include <common/dm_log.h>

#define __EXPOSED_KERNEL_FUNCTION(name, autores, ret, args) \
    dmk_##name##_t dmk_##name;                       \

#include <common/dm_kernel_funcs_list.h>

dmk_kallsyms_lookup_name_t find_lookup(void) {
    unsigned long kaddr = (unsigned long) &sprint_symbol;
    kaddr &= 0xffffffffff000000;

    char *fname_lookup = kzalloc(NAME_MAX, GFP_KERNEL);
    char *target_name = "kallsyms_lookup_name";

    dmk_kallsyms_lookup_name_t found;
    for (int i = 0x0 ; i < 0x4000000 ; i++ ) {
        sprint_symbol(fname_lookup, kaddr);

        if(strncmp(fname_lookup, target_name, strlen(target_name)) == 0) {
            dm_info("-- Located: 0x%px: %s\n", kaddr, fname_lookup);
            found = (dmk_kallsyms_lookup_name_t) kaddr;
            break;
        }

        kaddr += 0x4;
    }

    kfree(fname_lookup);
    return found;
}

#define __EXPOSED_KERNEL_FUNCTION_true(name, ret, args)                        \
    dm_info("- Resolving %s\n", #name);                                          \
    dmk_##name = (dmk_##name##_t) dmk_kallsyms_lookup_name(#name); \

#define __EXPOSED_KERNEL_FUNCTION_false(name, ret, args)

__nocfi void resolve_kernel_functions() {
    dm_info("Resolving kernel functions\n");

    dm_info("- Resolving kallsyms_lookup_name\n");
    dmk_kallsyms_lookup_name = find_lookup();
    if (dmk_kallsyms_lookup_name == NULL) {
        pr_err("Failed to find kallsyms_lookup_name\n");
        return;
    }

#define __EXPOSED_KERNEL_FUNCTION(name, autores, ret, args) __EXPOSED_KERNEL_FUNCTION_##autores(name, ret, args)
#include "dm_kernel_funcs_list.h"
}
