#ifndef DM_LOG_H
#define DM_LOG_H

#include <linux/printk.h>

#define pr_fmt(fmt) "droidmole: " fmt

#define dm_info(fmt, ...) \
	printk(KERN_INFO pr_fmt(fmt), ##__VA_ARGS__)

#endif //DM_LOG_H
