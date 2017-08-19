[![CircleCI](https://circleci.com/gh/SYNQfm/gosample.svg?style=svg&circle-token=a16459f5ae854e258ed0876ab8c4d1fdb14c7679)](https://circleci.com/gh/SYNQfm/gosample)
[![Coverage Status](https://coveralls.io/repos/github/SYNQfm/gosample/badge.svg?branch=master)](https://coveralls.io/github/SYNQfm/gosample?branch=master)

### Go sample app to use for new repos

This is a go sample repo.  This was adapted from [hydra](https://github.com/SYNQfm/hydra).

This sample app will implement the following

* Basic web server handling a /v1/status request
* Ability to communicate with Synq API
* DB Connection with setup
* Tests with mocking
* Marshalling struct to DB
* CircleCI setup
* Uses govendor to vendor/ management

This workspace works!  You can check out the test by running

`circleci build`

If you want to run it locally, you need to setup the test database

```
createuser -U <postgres admin> -h localhost -P -d gotest
createdb -U gotest -h localhost gosample_test
cd sql/environments/test && flyway migrate
```

### Setting up Cirlce CI

When you copy the workspace the first thing you should do is setup `circleci` and `coveralls`.  This is fairly straightforward.

* Go to [CircleCI Dashboard](https://circleci.com/projects/gh/SYNQfm)
* `Add Project`
* `Setup Project` the workspace you are working on
* `Starting Building` (assuming you select 2.0)
* Go to `Settings` and then [Status Badges](https://circleci.com/gh/SYNQfm/gosample/edit#badges) and copy the badge to your README.md
  * If its a private workspace, you need to setup an `API Token` in [API Permissions](https://circleci.com/gh/SYNQfm/gosample/edit#api)

### Setting up Coveralls

* Go to [Add Repos](https://coveralls.io/repos/new) to add your repo
* Find your repo (you may need to hit the `Synq Repos` button on the upper right)
* Click it "On"
* Go to [Details](https://coveralls.io/github/SYNQfm/gosample) and copy the `repo token`
* Go to CircleCI's [ENV settings]((https://circleci.com/gh/SYNQfm/gosample/edit#env-vars))  and create the ENV VAR `COVERALLS_TOKEN` using the copied repo token.
* Copy the Coveralls Badge and paste it into your README.md
* Once yuou have some tests, go to the coveralls [Settings](https://coveralls.io/github/SYNQfm/gosample/settings) page and change the Coverage Threshold to minimum `75%` and Decrease threshold to `5%`

### Setup your GitHub "protected" branch

* Go to GitHub repo [settings](https://github.com/SYNQfm/gosample/settings) and [branches](https://github.com/SYNQfm/gosample/settings/branches)
* Make `master` a protected branch
* Check `Project Branch`, `Require status checks to pass before merging` and make sure `ci/circleci` is required.  Save your changes
  * Note, coveralls won't show until you've ran a successful build with it so you can come back to make that required as well

### Things to change before you start using it

* Go to `.circleci/config.yml` and 
  * replace `gotest` with the user you want
  * replace `gosample_test` with the database you want
  * replace `gosample` with the name of your workspace
* Go to `sql/environments/test/flywayconf`
  * change the username/password for your app
  * change the database
* Change the `sql/migrations/V000__Init.sql` with the table you want

### Tools / Libraries Used

* Golang (duh!)
* Postgres
* Govendor `go get -u github.com/kardianos/govendor`
* CircleCi [cli](https://circleci.com/docs/2.0/local-jobs/)
