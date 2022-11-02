#ifndef DM_UTILS_H
#define DM_UTILS_H

#define __MAP0_BLOCK(m,...)
#define __MAP1_BLOCK(m,t,a,...) m(t,a);
#define __MAP2_BLOCK(m,t,a,...) m(t,a); __MAP1_BLOCK(m,__VA_ARGS__)
#define __MAP3_BLOCK(m,t,a,...) m(t,a); __MAP2_BLOCK(m,__VA_ARGS__)
#define __MAP4_BLOCK(m,t,a,...) m(t,a); __MAP3_BLOCK(m,__VA_ARGS__)
#define __MAP5_BLOCK(m,t,a,...) m(t,a); __MAP4_BLOCK(m,__VA_ARGS__)
#define __MAP6_BLOCK(m,t,a,...) m(t,a); __MAP5_BLOCK(m,__VA_ARGS__)
#define __MAP_BLOCK(n,...) __MAP##n##_BLOCK(__VA_ARGS__)

#endif //DM_UTILS_H
