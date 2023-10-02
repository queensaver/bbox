run_test:
	ng serve --host 0.0.0.0 --port 5050

update_cli:
	ng update @angular/cli

update_everything:
	ncu -u

tar:
	ng build
	tar cjf webapp.tar.bz2 dist/webapp
	cp webapp.tar.bz2 ../server
