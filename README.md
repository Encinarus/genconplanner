# Firebase Setup

Create an account on [Firebase](https://firebase.google.com/):

1. Click "Get Started"
2. Click "Add Project", and set a name. Click "Continue"
3. Decline Google Analytics, and click "Continue"
4. Click "Authentication", then "Get Started"
5. Click "Google" under Additional Providers; click the toggle to enable,
then select an email address, and click "Save"
6. Click the gear icon next to Project Overview, and select "Project
Settings"
7. Click the "Service Accounts" tab, then click "Create Service Account"
8. Select "Go" then click "Generate new private key"
9. Save the resulting JSON file
10. Click the "General" tab, then click the `</>` icon to set up a new web
app integration
11. Choose a name, then click "Register App". Copy the `const
firebaseConfig` that it generates for you

## Environment Variables

```
export FIREBASE_CONFIG=... # the contents of the JSON file from step 9, as a single line

# these values come from the const firebaseConfig object you generated in step 11
export FIREBASE_API_KEY=...
export FIREBASE_AUTH_DOMAIN=...
export FIREBASE_DATABASE_URL=...
export FIREBASE_PROJECT_ID=...
export FIREBASE_STORAGE_BUCKET=...
export FIREBASE_MESSAGING_SENDER_ID=...
```

Use [direnv](https://direnv.net/) to make this convenient so that you don't
have to re-export these values into your environment each time you work on
Gencon Planner


# Dev Env Setup

* Install [Docker Desktop](https://www.docker.com/products/docker-desktop/)
* Start the database in the background:
    $ docker compose up -d db
* Start the web server in the forgreound
    $ docker build --target web .
    $ docker compose up web

The web container depends upon the `update` container, which will download &
parse the events spreadsheet from gencon.com in the background.

To reset all state, use `docker compose down -v` to clear the DB data
volume, then follow the above steps again


## Alternate Dev Env

This setup uses a local postgres database and the `heroku` command. It may
be more like the true run-time environment than the Docker version above.

* Install Heroku
* Install Postgres following instructions here: https://devcenter.heroku.com/articles/heroku-postgresql#set-up-postgres-on-mac

To run server locally with Heroku, use `./build.sh && heroku local web`

To update the event listing locally, use `./build.sh && heroku local update`
