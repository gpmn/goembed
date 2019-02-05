集成七牛存储的qshell到go程序中。
goembed作为进程的一部分启动，接管进程的stdio.
通过telnet连上进程后，可以直接在进程的qshell内敲命令，手工调用函数等。
可以export更多的函数，协助分析问题。
目前对变量的支持不好，所以不推荐export变量。

