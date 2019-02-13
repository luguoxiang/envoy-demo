import subprocess
import os
import sys
import signal
import time

PORXY_UID=1337
PROXY_PORT=10000
PROXY_MANAGE_PORT=15000

def set_uid():
  try:    
    os.setuid(PORXY_UID)
  except BaseException as e:
    sys.stderr.write("Failed to setuid: %s\n" % str(e))
    sys.stdout.flush()
    sys.exit(1)

def sigterm_handler(signo, stack_frame):
  sys.stdout.write("shuting down proxy...\n")
  sys.stdout.write(subprocess.check_output(["sh", "iptable_clean.sh"]))
  sys.stdout.flush()
  sys.exit(0)

def get_env(key):
  value = os.getenv(key)
  if not value:
    sys.stderr.write("missing env %s" % key)
    sys.stdout.flush()
    sys.exit(1)
  return value
    
if len(sys.argv) < 2:
  sys.stderr.write("python run.py [envoy config path]\n")
  sys.stdout.flush()
  sys.exit(1)

config_path = sys.argv[1]

cluster_name = get_env("SERVICE_CLUSTER")
node_id = get_env("NODE_ID")

signal.signal(signal.SIGTERM, sigterm_handler)

iptable_env={
"PROXY_UID": "%d" % PORXY_UID,
"PROXY_PORT": "%d" % PROXY_PORT,
"PROXY_MANAGE_PORT": "%d" % PROXY_MANAGE_PORT,
"INBOUND_PORTS_INCLUDE":"9080",
}
sys.stdout.write (subprocess.check_output(["sh", "iptable_init.sh"], env=iptable_env))

cmd = [ "/usr/local/bin/envoy", "-c", config_path, "-l", "debug", "--service-node", node_id, "--service-cluster", cluster_name]
sys.stdout.write ("running %s\n" % cmd)
sys.stdout.flush()
 
p = subprocess.Popen( cmd ,preexec_fn=set_uid, stdout=subprocess.PIPE, stdin=subprocess.PIPE)
while True:
  line = p.stdout.readline()
  if line: 
    sys.stdout.write(line + "\n")
    sys.stdout.flush()
  time.sleep(1)
