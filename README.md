集成七牛存储的qshell到go程序中。
goembed作为进程的一部分启动，接管进程的stdio.
通过telnet连上进程后，可以直接在进程的qshell内敲命令，便于手工查看变量、调用函数等。
可以export更多的函数变量，协助分析问题。

