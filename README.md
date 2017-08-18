### Go sample app to use for new repos

This is a go sample repo.  This was adapted from [hydra](https://github.com/SYNQfm/hydra).

This sample app will implement the following

* Basic web server handling a /v1/status request
* Ability to communicate with Synq API
* DB Connection with setup
* Tests with mocking
* CircleCI setup
* Uses govendor to vendor/ management

This workspace works!  You can check out the test by running

`circleci build`

### Setting up Cirlce CI

When you copy the workspace the first thing you should do is setup `circleci` and `coveralls`.  This is fairly straightforward.

* Go to [CircleCI Dashboard](https://circleci.com/projects/gh/SYNQfm)
* `Add Project`
* `Setup Project` the workspace you are working on
* Copy the badge onto your README.md
* If its a private workspace, you

### Setting up Coveralls

* Go to [Coveralls Dashboard](https://coveralls.io/)

### Things to change before you start using it

* Go to `.circleci/config.yml` and 
  * replace `gotest` with the user you want
  * replace `go_test` with the database you want
  * replace `gosample` with the name of your workspace
* Go to `sql/environments/test/flywayconf`
  * change the username/password for your app
  * change the database
* Change the `sql/migrations/V000__Init.sql` with the table you want
