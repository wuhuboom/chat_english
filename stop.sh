ps -ef|grep "main"
kill -9 $(pidof 'main')