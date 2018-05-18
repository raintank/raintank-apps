#!/usr/bin/env python2.7
import mysql.connector
import json
import time
import socket
import requests
from time import sleep
import argparse
import tempfile
import os
import logging

# Outputs stats for raintank-apps database
#
# NS1 Summary
#   Tasks defined total
#   Tasks enabled
#   Tasks disabled
#   Number of org_id's
#   Number of api_keys
#   Domains per api_key
#   Orgs with duplicate api_key
# NS1 Org Summary
#   Domain count
#   Number of API keys
#   domain names
#   customer - external ldap to fetch this?
# NS1 API_KEY Summary
#   Domains
#   org_id's using key (number)

# globals
#

# overall stats
APP_STATS = {
    'ENABLED': 0,
    'DISABLED': 0,
    'TOTAL': 0
}

# NS1 specific stats
NS1_STATS = {
    'ENABLED': 0,
    'DISABLED': 0,
    'TOTAL': 0
}

NS1_ORG_IDS = {}
NS1_KEYS = {}
NS1_DOMAINS = {}
NS1_DOMAINS_BY_ORG_ID = {}
NS1_DOMAINS_BY_API_KEY = {}

# Voxter specific stats
VOXTER_STATS = {
    'ENABLED': 0,
    'DISABLED': 0,
    'TOTAL': 0
}
VOXTER_ORG_IDS = {}
VOXTER_KEYS = {}

# Not working
GITSTATS_STATS = {
    'ENABLED': 0,
    'DISABLED': 0,
    'TOTAL': 0
}
# Should not have any of those be positive
UNKNOWN_PLUGIN_STATS = {
    'ENABLED': 0,
    'DISABLED': 0,
    'TOTAL': 0
}


def setup_logging():
    logger = logging.getLogger('generate_stats.py')
    logger.setLevel(logging.DEBUG)
    fh = logging.FileHandler('/tmp/generate_stats.log')
    fh.setLevel(logging.DEBUG)
    # create console handler and set level to debug
    ch = logging.StreamHandler()
    ch.setLevel(logging.DEBUG)
    # create formatter
    formatter = logging.Formatter(
        '%(asctime)s - %(name)s - %(levelname)s - %(message)s')
    # add formatter to ch
    ch.setFormatter(formatter)
    fh.setFormatter(formatter)
    # add ch to logger
    logger.addHandler(ch)
    logger.addHandler(fh)
    return logger


def add_row(row):
    global logger
    global APP_STATS
    global NS1_STATS
    global NS1_ORG_IDS
    global NS1_KEYS
    global NS1_DOMAINS_BY_ORG_ID
    global NS1_DOMAINS_BY_API_KEY
    global VOXTER_STATS
    global VOXTER_ORG_IDS
    global VOXTER_KEYS
    global GITSTATS_STATS
    global UNKNOWN_PLUGIN_STATS
    global ORG_ID_TO_SLUG

    row_id = row[0]
    task_name = row[1]
    task_config = row[2]
    task_interval = row[3]
    org_id = str(row[4])
    if org_id not in ORG_ID_TO_SLUG:
        logger.info("getting slug for id {}".format(org_id))
        org_slug = get_org_slug(org_id)
        ORG_ID_TO_SLUG[org_id] = org_slug
    task_enabled = row[5]
    task_route = row[6]
    task_created = row[7]
    task_updated = row[8]
    APP_STATS['TOTAL'] += 1
    if task_enabled == 1:
        APP_STATS['ENABLED'] += 1
    else:
        APP_STATS['DISABLED'] += 1
    tc = json.loads(task_config)
    for plugin_type in tc:
        if plugin_type == '/raintank/apps/ns1':
            ns1_key = tc[plugin_type].get('ns1_key').strip()
            zone_name = tc[plugin_type].get('zone')
            if zone_name is not None:
                zone = zone_name.strip()
                if zone not in NS1_DOMAINS_BY_ORG_ID:
                    NS1_DOMAINS_BY_ORG_ID[zone] = org_id
                if zone not in NS1_DOMAINS_BY_API_KEY:
                    NS1_DOMAINS_BY_API_KEY[zone] = ns1_key
            if ns1_key in NS1_KEYS:
                NS1_KEYS[ns1_key] += 1
            else:
                NS1_KEYS[ns1_key] = 1
            if org_id in NS1_ORG_IDS:
                NS1_ORG_IDS[org_id] += 1
            else:
                NS1_ORG_IDS[org_id] = 1
            NS1_STATS['TOTAL'] += 1
            if task_enabled == 1:
                NS1_STATS['ENABLED'] += 1
            else:
                NS1_STATS['DISABLED'] += 1
        elif plugin_type == '/raintank/apps/voxter':
            logger.debug(tc[plugin_type])
            voxter_key = tc[plugin_type].get('voxter_key').strip()
            if voxter_key in VOXTER_KEYS:
                VOXTER_KEYS[voxter_key] += 1
            else:
                VOXTER_KEYS[voxter_key] = 1
            if org_id in VOXTER_ORG_IDS:
                VOXTER_ORG_IDS[org_id] += 1
            else:
                VOXTER_ORG_IDS[org_id] = 1
            VOXTER_STATS['TOTAL'] += 1
            if task_enabled == 1:
                VOXTER_STATS['ENABLED'] += 1
            else:
                VOXTER_STATS['DISABLED'] += 1
        elif plugin_type == '/raintank/apps/gitstats':
            GITSTATS_STATS['TOTAL'] += 1
        else:
            logger.warning("found unknown plugin: {}"
                           .format(plugin_type))
            UNKNOWN_PLUGIN_STATS['TOTAL'] += 1


def app_summary(timestamp):
    global logger
    global APP_STATS
    logger.info("Summary of APP")
    logger.info("APP TASKS {} ENABLED {} DISABLED {}"
                .format(APP_STATS['TOTAL'],
                        APP_STATS['ENABLED'],
                        APP_STATS['DISABLED']))
    metrics = [
        'raintank.apps.summary.tasks.total;app=raintank-apps {} {}'.format(
            APP_STATS['TOTAL'], timestamp),
        'raintank.apps.summary.tasks.enabled;app=raintank-apps {} {}'.format(
            APP_STATS['ENABLED'], timestamp),
        'raintank.apps.summary.tasks.disabled;app=raintank-apps {} {}'.format(
            APP_STATS['DISABLED'], timestamp)
        ]
    publish_stats(metrics)


def ns1_summary(timestamp):
    global logger
    global NS1_STATS
    logger.info('Summary of NS1')
    logger.info("NS1 TASKS {} ENABLED {} DISABLED {}"
                .format(NS1_STATS['TOTAL'],
                        NS1_STATS['ENABLED'],
                        NS1_STATS['DISABLED']))
    metrics = [
        'raintank.apps.plugin.ns1.total;app=raintank-apps {} {}'.format(
            NS1_STATS['TOTAL'], timestamp),
        'raintank.apps.plugin.ns1.enabled;app=raintank-apps {} {}'.format(
            NS1_STATS['ENABLED'], timestamp),
        'raintank.apps.plugin.ns1.disabled;app=raintank-apps {} {}'.format(
            NS1_STATS['DISABLED'], timestamp)
    ]
    # per org_id n1 stats
    for an_org in NS1_ORG_IDS:
        logger.info("NS1: ORG_ID {} domain_count: {}"
                    .format(an_org, NS1_ORG_IDS[an_org]))
        org_slug = ORG_ID_TO_SLUG[an_org]
        logger.info('raintank.apps.plugin.ns1.domain_count;app=raintank-apps;org_id={};org_slug={} {} {}'
                    .format(an_org, org_slug, NS1_ORG_IDS[an_org], timestamp))
        metrics.append(
            'raintank.apps.plugin.ns1.domain_count;app=raintank-apps;org_id={};org_slug={} {} {}'.format(
                an_org, org_slug, NS1_ORG_IDS[an_org], timestamp)
        )
    for domain in NS1_DOMAINS_BY_ORG_ID:
        metrics.append(
            'raintank.apps.plugin.ns1.domains.org_id;app=raintank-apps;domain=\"{}\" {} {}'.format(
                domain, NS1_DOMAINS_BY_ORG_ID[domain], timestamp)
        )
    # per ns1_key stats
    for api_key in NS1_KEYS:
        logger.info("NS1: API_KEY {} domain_count: {}"
                    .format(api_key, NS1_KEYS[api_key]))
        metrics.append(
            'raintank.apps.plugin.ns1.api_key.domain_count;app=raintank-apps;api_key=\"{}\" {} {}'.format(
                api_key, NS1_KEYS[api_key], timestamp)
        )
    for domain in NS1_DOMAINS_BY_API_KEY:
        metrics.append(
            'raintank.apps.plugin.ns1.domains.api_key;app=raintank-apps;domain=\"{}\";api_key=\"{}\" {} {}'.format(
                domain, NS1_DOMAINS_BY_API_KEY[domain], 1, timestamp)
        )
    publish_stats(metrics)


def voxter_summary(timestamp):
    global logger
    global VOXTER_STATS
    logger.info("VOXTER TASKS {} ENABLED {} DISABLED {}"
                .format(VOXTER_STATS['TOTAL'],
                        VOXTER_STATS['ENABLED'],
                        VOXTER_STATS['DISABLED']))
    metrics = [
        'raintank.apps.plugin.voxter.total;app=raintank-apps {} {}'.format(
            VOXTER_STATS['TOTAL'], timestamp),
        'raintank.apps.plugin.voxter.enabled;app=raintank-apps {} {}'.format(
            VOXTER_STATS['ENABLED'], timestamp),
        'raintank.apps.plugin.voxter.disabled;app=raintank-apps {} {}'.format(
            VOXTER_STATS['DISABLED'], timestamp)
    ]
    # per org_id voxter stats
    for an_org in VOXTER_ORG_IDS:
        logger.info("VOXTER: ORG_ID {} domain_count: {}"
                    .format(an_org, VOXTER_ORG_IDS[an_org]))
        metrics.append(
            'raintank.apps.plugin.voxter.org_id.domain_count;app=raintank-apps;org_id={} {} {}'
            .format(an_org, VOXTER_ORG_IDS[an_org], timestamp)
        )
    # per voxter_key stats
    for api_key in VOXTER_KEYS:
        logger.info("VOXTER: API_KEY {} domain_count: {}"
                    .format(api_key, VOXTER_KEYS[api_key]))
        metrics.append(
            'raintank.apps.plugin.voxter.api_key.domain_count;app=raintank-apps;api_key=\"{}\" {} {}'
            .format(api_key, VOXTER_KEYS[api_key], timestamp)
        )
    publish_stats(metrics)


def gitstats_summary(timestamp):
    global logger
    global GITSTATS_STATS
    logger.info("GITSTATS TASKS {} ENABLED {} DISABLED {}"
                .format(GITSTATS_STATS['TOTAL'],
                        GITSTATS_STATS['ENABLED'],
                        GITSTATS_STATS['DISABLED']))


def unknown_summary(timestamp):
    global logger
    global UNKNOWN_PLUGIN_STATS
    logger.info("UNKNOWN TASKS {} ENABLED {} DISABLED {}"
                .format(UNKNOWN_PLUGIN_STATS['TOTAL'],
                        UNKNOWN_PLUGIN_STATS['ENABLED'],
                        UNKNOWN_PLUGIN_STATS['DISABLED']))


def get_org_slug(org_id):
    global logger
    org_slug = "unknown"
    logger.info("getting orgSlug for id {}".format(org_id))
    try:
        r = requests.get("https://grafana.com/api/orgs/{}/members".format(org_id))
        if r.status_code == 200:
            org_slug = r.json()['items'][0]['orgSlug']
    except:
        logger.info("unable to get org slug for org_id {}".format(org_id))
    return org_slug


def publish_stats(metrics):
    global logger
    global GRAPHITE_HOST
    global GRAPHITE_PORT
    logger.info("Publishing... {}:{}".format(GRAPHITE_HOST, GRAPHITE_PORT))
    conn = socket.socket(socket.AF_INET, socket.SOCK_STREAM)
    conn.settimeout(60)
    addr = (GRAPHITE_HOST, GRAPHITE_PORT)
    try:
        # send it
        conn.connect(addr)
        for metric in metrics:
            message = metric + '\n'
            resp = conn.sendall(message.encode("ascii"))
        conn.close()
        # not too fast...
        sleep(0.25)
    except socket.timeout:
        logger.error("Took over 60 second(s) to connect to {}"
                     .format(GRAPHITE_HOST))
    except socket.gaierror:
        logger.error("No address associated with hostname {}"
                     .format(GRAPHITE_HOST))
    except Exception as error:
        logger.error("unknown exception while connecting to {} - {}"
                     .format(GRAPHITE_HOST, error))


def read_org_id_cache(filename):
    global logger
    logger.info("reading cache {}".format(filename))
    data = {}
    if os.path.isfile(filename):
        with open(filename) as json_file:
            logger.info("reading cache")
            data = json.load(json_file)
    return data


def write_org_id_cache(filename, data):
    global logger
    logger.info("writing cache {}".format(filename))
    with open(filename, 'w') as outfile:
        json.dump(data, outfile)


def generate_stats(sqlhost, sqlport, sqluser, sqlpass):
    global logger
    start_time = int(time.time())
    try:
        conn = mysql.connector.connect(
            host=sqlhost,
            database='task_server',
            user=sqluser,
            password=sqlpass,
            port=sqlport)
        cursor = conn.cursor()
        cursor.execute("SELECT * FROM task_server.task;")
        rows = cursor.fetchall()
        logger.info('Total Row(s): {}'.format(cursor.rowcount))
        for row in rows:
            add_row(row)
        ns1_summary(start_time)
        voxter_summary(start_time)
        gitstats_summary(start_time)
        unknown_summary(start_time)
        app_summary(start_time)
    except Exception as e:
        logger.error(e)
    finally:
        cursor.close()
        conn.close()


if __name__ == '__main__':
    """
    run script
    """
    parser = argparse.ArgumentParser(
        description='Send raintank-app database stats to graphite',
        formatter_class=lambda prog: argparse.HelpFormatter(
            prog,
            max_help_position=90,
            width=110),
        add_help=True)

    parser.add_argument('--sqlhost', help='MySQL hostname',
                        action='store', default="localhost")
    parser.add_argument('--sqlport', help='MySQL port',
                        action='store', default=3306)
    parser.add_argument('--sqluser', help='MySQL username',
                        action='store', default='root')
    parser.add_argument('--sqlpassword', help='MySQL password',
                        action='store', default='password')

    # graphite destination
    parser.add_argument('--graphite-host', help='graphite hostname',
                        action='store', default='localhost')
    parser.add_argument('--graphite-port', help='graphite port',
                        action='store', default=2003)

    args = parser.parse_args()
    logger = setup_logging()
    # read cache if it exists
    ORG_ID_TO_SLUG = read_org_id_cache('/tmp/org_id_cache.json')
    GRAPHITE_HOST = args.graphite_host
    GRAPHITE_PORT = args.graphite_port
    logger.info("Pulling stats from database")
    generate_stats(
        args.sqlhost,
        args.sqlport,
        args.sqluser,
        args.sqlpassword)
    # write org_ids back to cache
    write_org_id_cache('/tmp/org_id_cache.json', ORG_ID_TO_SLUG)
