.PHONY: gen-mock

gen-mock:
	@echo "generating mysql client mock file..."
	mockgen -source=client.go -destination=mockmysql/mysql_mock.go -package=mockmysql
	@echo "generate mysql client mock file done"
