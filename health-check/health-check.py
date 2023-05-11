import requests
import logging
import os

logging.basicConfig(
    format='%(asctime)s %(levelname)-8s %(message)s',
    level=logging.INFO,
    datefmt='%Y-%m-%d %H:%M:%S')

log = logging.getLogger("deployment-controller-health-check")

def check_status_page(url, cookies, release_name):

    log.info(f"Checking {url} to see if {release_name} is on the status page. Using the cookies {cookies}.")

    resp = requests.get(url, cookies=cookies)

    if release_name in resp.text:
        log.info(f"{resp.text} does contain {release_name}!")
        return True
    else:
        log.info(f"{resp.text} does not contain {release_name}!")
        return False

def validate_env(cookie_releases, cookie_names, cookie_vals):
    if(len(cookie_releases) == len(cookie_names) == len(cookie_vals)):
        log.info("The environment variables are all the proper length.")
        return True
    else:
        log.info("The environment variables are not equal length. They must be space delimited. Check the pod env variables.")
        return False

def run_health_check():
    if(validate_env(cookie_releases, cookie_names, cookie_vals) is False):
        return 500

    for i in range(len(cookie_releases)):
        cookies = { cookie_names[i] : cookie_vals[i] }
        
        if(check_status_page(url, cookies, cookie_releases[i]) is False):
            return 500
    
    return 200

# envs are colon delimited. see cronjob-health-check.yaml in deployment-controller's tempate for how they're inserted.
# remove empty indices with trailing semi colons from the envs
cookie_releases = os.getenv("cookie_releases").split(";")[:-1]
cookie_names = os.getenv("cookie_names").split(";")[:-1]
cookie_vals = os.getenv("cookie_vals").split(";")[:-1]
url = os.getenv("url")

print(run_health_check())
