# dbcmp

Introducing `dbcmp` - Your Command Line Database Content Comparison Tool

## Description

`dbcmp` is a powerful and efficient command line tool designed to simplify the process of comparing content between two databases. `dbcmp` empowers you to quickly identify discrepancies and inconsistencies in data, facilitating data integrity and accuracy.

## Key Features

1. `dbcmp` comes with a user-friendly command line interface that requires minimal configuration.
2. Currently, `dbcmp` supports MySQL and PostgreSQL. You can also compare content between same database systems.
3. You can specify the tables (and maybe columns) you want to compare, giving you granular control over the comparison process and avoiding unnecessary comparisons. (TBA)
4. The tool provides minimal output, allowing you to easily focus on which tables are different.
5. Support for large databases; native pagination provides a comparison process that doesn't bring much load on your databases.

## How to Use dbcmp

1. To install dbcmp, simply run:

   ```sh
   go install github.com/mattermost/dbcmp/cmd/dbcmp
   ```

2. To Configure, dbcmp requires access credentials for the databases you want to compare. You can set them directly as command line arguments.

3. To perform a basic comparison between two databases, use the following command:

   ```sh
   dbcmp --source source_dsn --target target_dsn
   ```

4. If you wish to exclude specific tables, you can use the `--exclude` option:

   ```sh
   dbcmp --source source_dsn --target target_dsn --exclude=table1,table2
   ```

5. If you would like to change page size, you can use `--page-size` option to optimize your comparison performance:

   ```sh
   bcmp --source source_dsn --target target_dsn --page-size=5000
   ```

Now you have the power to compare database content effortlessly with dbcmp. Happy comparing!

## LICENSE

[MIT](LICENSE)
