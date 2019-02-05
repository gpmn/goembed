集成七牛存储的qshell到go程序中，目前只在ubuntu上测试过。  
goembed作为进程的一部分启动，接管进程的stdio.  
通过telnet连上进程后，可以直接在进程的qshell内敲命令，手工调用函数等。  
可以export更多的函数，协助分析问题。  
目前对变量的支持不好，所以不推荐export变量。  

之前还有一个goinside，还能针对C代码进行调试，现在已经废止了。

先go run test/main.go，然后再另一个console中telnet 127.0.0.1 6666   
  `
  ubuntu@localhost:~$ telnet 127.0.0.1 6666  
  Trying 127.0.0.1...  
  Connected to 127.0.0.1.  
  Escape character is '^]'.  
                    
  \>\>\> ExportTest("beauty")  
  Hello beauty  
  good!  
  \>\>\> os.Exit(1)  
  Connection closed by foreign host.  
`
