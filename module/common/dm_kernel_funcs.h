#ifndef DM_KERNEL_FUNCS_H
#define DM_KERNEL_FUNCS_H

#define __EXPOSED_KERNEL_FUNCTION(name, autores, ret, args) \
    typedef ret (* dmk_##name##_t) args ;                   \
    extern dmk_##name##_t dmk_##name;                       \

#include <common/dm_kernel_funcs_list.h>

void resolve_kernel_functions(void);

#endif //DM_KERNEL_FUNCS_H
