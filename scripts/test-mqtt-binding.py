#!/usr/bin/env python3
"""Wire-level MQTT acceptance checks against an isolated running Compose stack.
No third-party Python packages. Example:
  python3 scripts/test-mqtt-binding.py --api http://localhost:18070 --port 11883
Set --compose-project to additionally exercise broker/API restarts and outages.
"""
import argparse
import concurrent.futures
import json
import os
import socket
import struct
import subprocess
import time
import urllib.request
import urllib.error
import uuid


def varint(value):
    out = bytearray()
    while True:
        digit = value % 128
        value //= 128
        out.append(digit | (128 if value else 0))
        if not value:
            return bytes(out)


def string(value):
    raw = value.encode()
    return struct.pack('!H', len(raw)) + raw


class MQTT:
    def __init__(self, port, version=5, client=None, clean=True, will=None, username=None, password=None):
        self.version = version
        self.sock = socket.create_connection(('127.0.0.1', port), timeout=4)
        flags = 2 if clean else 0
        if will:
            flags |= 4
        if username:
            flags |= 192
        payload = string(client or str(uuid.uuid4()))
        props = b'\x11' + struct.pack('!I', 300) if not clean else b''
        header = string('MQTT') + bytes([version, flags]) + struct.pack('!H', 30)
        if version == 5:
            header += varint(len(props)) + props
        if will:
            if version == 5:
                payload += b'\0'
            payload += string(will[0]) + string(will[1])
        if username:
            payload += string(username) + string(password)
        self.send(0x10, header + payload)
        kind, ack = self.packet()
        assert kind == 0x20 and ack[1] == 0, (kind, ack)
        self.session_present = bool(ack[0])

    def send(self, kind, body=b''):
        self.sock.sendall(bytes([kind]) + varint(len(body)) + body)

    def recv(self, n):
        out = b''
        while len(out) < n:
            chunk = self.sock.recv(n-len(out))
            if not chunk:
                raise EOFError('broker closed connection')
            out += chunk
        return out

    def packet(self, timeout=4):
        self.sock.settimeout(timeout)
        kind = self.recv(1)[0]
        length, shift = 0, 0
        while True:
            b = self.recv(1)[0]
            length += (b & 127) << shift
            shift += 7
            if b < 128:
                break
        return kind, self.recv(length)

    def subscribe(self, topic, qos=2, mid=1):
        self.send(0x82, struct.pack('!H', mid) + (b'\0' if self.version == 5 else b'') + string(topic) + bytes([qos]))
        kind, body = self.packet()
        assert kind == 0x90, (kind, body)
        return body[-1]

    def publish(self, topic, payload, qos=1, key=None, media='application/json', mid=7, dup=False, retain=False, extra=None, wait=True, raw_properties=b''):
        props = b''
        if media is not None:
            props += b'\x03' + string(media)
        if key is not None:
            props += b'\x26' + string('idempotency-key') + string(key)
        for name, value in extra or []:
            props += b'\x26' + string(name) + string(value)
        props += raw_properties
        body = string(topic) + (struct.pack('!H', mid) if qos else b'')
        if self.version == 5:
            body += varint(len(props)) + props
        self.send(0x30 | (qos << 1) | (8 if dup else 0) | int(retain), body + payload)
        if qos and wait:
            return self.packet()
        return None

    def next_delivery(self, timeout):
        while True:
            kind, body = self.packet(timeout)
            if kind == 0x62:
                self.send(0x70, body[:2])
                continue
            return kind, body

    def resource(self, timeout=4):
        kind, body = self.next_delivery(timeout)
        assert kind >> 4 == 3, (kind, body)
        length = struct.unpack('!H', body[:2])[0]
        topic = body[2:2+length].decode()
        pos = 2+length
        qos = (kind >> 1) & 3
        if qos:
            mid = body[pos:pos+2]
            pos += 2
            self.send(0x50 if qos == 2 else 0x40, mid)
        props = b''
        if self.version == 5:
            plen, shift = 0, 0
            while True:
                b = body[pos]
                pos += 1
                plen += (b & 127) << shift
                shift += 7
                if b < 128:
                    break
            props = body[pos:pos+plen]
            pos += plen
        assert b'csapi-internal-provenance' not in props
        assert b'idempotency-key' not in props
        assert not kind & 1, 'retained message delivered'
        return topic, json.loads(body[pos:]), props

    def quiet(self, timeout=.25):
        try:
            packet = self.next_delivery(timeout)
        except (socket.timeout, EOFError):
            return
        raise AssertionError(('unexpected delivery', packet))

    def close(self, abrupt=False):
        if not abrupt:
            try:
                self.send(0xE0)
            except OSError:
                pass
        self.sock.close()


def main():
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--api', default='http://localhost:18070')
    parser.add_argument('--port', type=int, default=11883)
    parser.add_argument('--compose-project')
    parser.add_argument('--username', default=os.getenv('MQTT_CLIENT_USERNAME'))
    parser.add_argument('--password', default=os.getenv('MQTT_CLIENT_PASSWORD'))
    args = parser.parse_args()
    token = str(uuid.uuid4())
    resources = []

    def client(version=5, **kw):
        return MQTT(args.port, version, username=args.username, password=args.password, **kw)

    def http(path, payload=None, method=None):
        req = urllib.request.Request(args.api+path, data=None if payload is None else json.dumps(payload).encode(), headers={'Content-Type': 'application/json'}, method=method)
        with urllib.request.urlopen(req, timeout=8) as res:
            body = res.read()
            return res.status, res.headers, json.loads(body) if body else None

    def system(name):
        return {'type': 'Feature', 'geometry': None, 'properties': {'uid': 'urn:uuid:'+str(uuid.uuid4()), 'name': name, 'featureType': 'http://www.w3.org/ns/sosa/Sensor', 'assetType': 'Equipment'}}

    def created(payload, path='/systems'):
        status, headers, _ = http(path, payload)
        assert status == 201
        identity = headers['Location'].rsplit('/', 1)[-1]
        if path == '/systems':
            resources.append(identity)
        return identity

    def rejected(topic, payload=b'{}', **kw):
        c = client(kw.pop('version', 5))
        try:
            packet = c.publish(topic, payload, **kw)
            assert packet and packet[0] >> 4 in (4, 5, 14), packet
            if packet[0] >> 4 != 14:
                assert len(packet[1]) >= 3 and packet[1][2] >= 128, packet
        except (EOFError, ConnectionResetError, BrokenPipeError):
            pass
        finally:
            c.close()

    if args.username:
        try:
            anonymous = MQTT(args.port)
        except (AssertionError, EOFError):
            pass
        else:
            anonymous.close()
            raise AssertionError('authenticated profile accepted anonymous login')
    sub5, sub3 = client(), client(4)
    assert sub5.subscribe('node1/#') == 2
    assert sub3.subscribe('node1/#') == 2
    bad = client()
    for topic in ['node1/observations:data', 'node1/systems:data/xml', 'node2/#', 'node1/systems:events', '$share/group/node1/#']:
        assert bad.subscribe(topic, mid=11) >= 128, topic
    bad.close()

    # Publisher reconnect backoff is one second after a broker-only restart.
    time.sleep(1.2)
    # Subscriptions precede creation: every future hierarchical topic is covered.
    root = created(system('HTTP '+token))
    for c in [sub5, sub3]:
        received = [c.resource() for _ in range(3)]
        assert {m[0] for m in received} == {'node1/systems:data', 'node1/systems:data/sml-json', 'node1/systems/'+root}
        event = next(m[1] for m in received if m[0].endswith(root))
        assert event['type'] == 'org.ogc.api.csapi.system.create'
        assert event['subject'] == args.api+'/systems/'+root
        if c.version == 5:
            assert all(m[2] for m in received), 'missing MQTT 5 Content Type'
    sub3.close()

    # Canonicalization, v3 client input to v5 output, and representation fan-out.
    topic = 'node1/systems/'+root+'/subsystems:data'
    payload = json.dumps(system('MQTT child '+token)).encode()
    v3 = client(4)
    assert v3.publish(topic, payload)[0] == 0x40
    received = [sub5.resource() for _ in range(3)]
    msg = next(m[1] for m in received if m[0] == topic)
    child = msg['id']
    assert msg['properties']['name'].startswith('MQTT child')
    assert http('/systems/'+child)[0] == 200
    event = next(m[1] for m in received if m[0].endswith('/'+child))
    assert event['resourceparent'] == args.api+'/systems/'+root
    v3.close()

    # Concurrent, reconnect-safe retries with changed client IDs; original canonical bytes.
    keyed = json.dumps(system('keyed '+token)).encode()
    def retry(i):
        c = client(client='retry-'+str(i)+'-'+token)
        ack = c.publish('node1/systems:data', keyed, key='key-'+token, media='application/geo+json')
        assert ack[0] == 0x40 and (len(ack[1]) < 3 or ack[1][2] < 128), ack
        c.close()
    with concurrent.futures.ThreadPoolExecutor(max_workers=8) as pool:
        list(pool.map(retry, range(8)))
    received = [sub5.resource() for _ in range(10)]
    canonical = [m[1] for m in received if m[0] == 'node1/systems:data']
    assert len(canonical) == 8 and all(v == canonical[0] for v in canonical)
    keyed_id = canonical[0]['id']
    resources.append(keyed_id)
    assert sum(m[0] == 'node1/systems/'+keyed_id for m in received) == 1
    assert sum(m[0] == 'node1/systems:data/sml-json' for m in received) == 1
    rejected('node1/systems:data', json.dumps(system('conflict')).encode(), key='key-'+token, media='application/geo+json')
    sub5.quiet()

    # Public PUBLISH properties survive canonicalization; private metadata does not.
    correlation=b'\x00opaque\xff'
    metadata=(b'\x01\x01' + b'\x02'+struct.pack('!I',120)
              + b'\x08'+string('reply/metadata-check')
              + b'\x09'+struct.pack('!H',len(correlation))+correlation
              + b'\x26'+string('trace-test')+string('preserved'))
    writer=client()
    ack=writer.publish('node1/systems:data',json.dumps(system('metadata '+token)).encode(),
                       key='metadata-'+token,media='application/geo+json',raw_properties=metadata)
    assert ack[0]==0x40 and (len(ack[1])<3 or ack[1][2]<128),ack
    received=[sub5.resource() for _ in range(3)]
    _,resource,props=next(m for m in received if m[0]=='node1/systems:data')
    resources.append(resource['id'])
    assert b'\x01\x01' in props and b'reply/metadata-check' in props
    assert correlation in props and b'trace-test' in props and b'preserved' in props
    writer.close()

    # Incoming QoS 2 retransmission reuses its accepted operation.
    q2 = client()
    q2body = json.dumps(system('qos2 '+token)).encode()
    for dup in [False, True]:
        ack = q2.publish('node1/systems:data', q2body, qos=2, key='qos2-'+token, media='application/geo+json', mid=42, dup=dup)
        assert ack[0] == 0x50 and (len(ack[1]) < 3 or ack[1][2] < 128), ack
    for _ in range(2):
        q2.send(0x62, struct.pack('!H', 42))
        assert q2.packet()[0] == 0x70
    received = [sub5.resource() for _ in range(3)]
    q2id = next(m[1]['id'] for m in received if m[0] == 'node1/systems:data')
    resources.append(q2id)
    sub5.quiet()
    # Reused packet identifier with another key is a new operation.
    nextbody = json.dumps(system('packet reuse '+token)).encode()
    assert q2.publish('node1/systems:data', nextbody, qos=2, key='next-'+token, media='application/geo+json', mid=42)[0] == 0x50
    q2.send(0x62, struct.pack('!H', 42)); assert q2.packet()[0] == 0x70
    received = [sub5.resource() for _ in range(3)]
    nextid = next(m[1]['id'] for m in received if m[0] == 'node1/systems:data')
    assert nextid != q2id
    resources.append(nextid)
    q2.close()

    # QoS1 duplicate PUBLISH with the same key reuses only the original stream.
    again = client()
    for dup in [False, True]:
        assert again.publish('node1/systems:data', q2body, key='qos2-'+token,
                             media='application/geo+json', mid=99, dup=dup)[0] == 0x40
        _, replay, _ = sub5.resource()
        assert replay['id'] == q2id
    again.close()
    sub5.quiet()

    # Alternate SensorML input at QoS0 still produces the same logical streams.
    sml = {'type':'PhysicalSystem', 'label':'SML '+token,
           'uniqueId':'urn:uuid:'+str(uuid.uuid4()),
           'definition':'http://www.w3.org/ns/sosa/Sensor'}
    writer = client()
    writer.publish('node1/systems:data/sml-json', json.dumps(sml).encode(), qos=0,
                   key='sml-'+token, media='application/sml+json')
    received = [sub5.resource() for _ in range(3)]
    data = next(m[1] for m in received if m[0] == 'node1/systems:data')
    resources.append(data['id'])
    native = next(m[1] for m in received if m[0].endswith('/sml-json'))
    assert native['label'] == sml['label'] and native['id'] == data['id']
    writer.close()

    # No unvalidated input, missing/conflicting media, metadata spoofing, or retained commands.
    rejected('node1/systems:data', b'{broken', media='application/geo+json')
    rejected('node1/systems:data', payload, media=None)
    rejected('node1/systems:data', payload, media='application/json')
    rejected('node1/systems:data', payload, media='application/geo+json', key='')
    rejected('node1/systems:data', payload, media='application/geo+json', key='a', extra=[('idempotency-key', 'b')])
    rejected('node1/systems:data', payload, media='application/geo+json', extra=[('csapi-internal-provenance', 'forged')])
    rejected('node1/systems/'+root, payload, media='application/cloudevents+json')
    rejected('node1/systems:data', payload, media='application/geo+json', qos=2)
    rejected('node1/systems:data', payload, version=4, qos=2)
    rejected('node1/systems:data', payload, media='application/geo+json', retain=True)
    rejected('node1/systems:data', b' ' * (10*1024*1024+1), media='application/geo+json')
    # The embedded endpoint refuses Will configuration at CONNECT.
    try:
        will = client(will=('node1/systems:data', json.dumps(system('must not escape'))))
    except (AssertionError, EOFError, ConnectionResetError):
        pass
    else:
        will.close(abrupt=True)
        raise AssertionError('endpoint accepted a Last Will')
    sub5.quiet(.5)

    # Exercise both directions for every eligible Part 2 resource on real wires.
    # Exact subscriptions keep batch notifications on their own collection topic.
    sub5.close()
    sub5=client()
    prefix='node1/systems/'+root+'/subsystems/'+child
    assert sub5.subscribe(prefix)==2
    assert sub5.subscribe('node1/systems/'+keyed_id)==2

    def both_directions(kind, topic, post_path, item_prefix, payload):
        watchers = [client(version) for version in [5, 4]]
        for watcher in watchers:
            assert watcher.subscribe(topic) == 2
        writer=client()
        raw=json.dumps(payload).encode()
        ack=writer.publish(topic,raw,key=kind+'-'+token)
        assert ack[0]==0x40 and (len(ack[1])<3 or ack[1][2]<128), (kind,ack)
        identity=None
        for watcher in watchers:
            actual,resource,props=watcher.resource()
            assert actual==topic and resource.get('id'), (kind,resource)
            if watcher.version == 5:
                assert props
            if identity is None:
                identity=resource['id']
            assert identity==resource['id']
        assert http(item_prefix+identity)[0]==200
        http_payload=dict(payload)
        if 'uid' in http_payload:
            http_payload['uid']='urn:uuid:'+str(uuid.uuid4())
        status,headers,_=http(post_path,http_payload)
        assert status==201
        for watcher in watchers:
            _,from_http,_=watcher.resource()
            assert from_http['id']==headers['Location'].rsplit('/',1)[-1]
            assert from_http['id']!=identity
        # Every resource also enters through MQTT 3.1.1, with media inferred
        # from its topic, and reaches subscribers using both protocol versions.
        v3_payload=dict(payload)
        if 'uid' in v3_payload:
            v3_payload['uid']='urn:uuid:'+str(uuid.uuid4())
        v3=client(4)
        ack=v3.publish(topic,json.dumps(v3_payload).encode())
        assert ack[0]==0x40, (kind,ack)
        v3_identity=None
        for watcher in watchers:
            actual,resource,_=watcher.resource()
            assert actual==topic and resource['id']!=identity
            if v3_identity is None:
                v3_identity=resource['id']
            assert v3_identity==resource['id']
            watcher.quiet();watcher.close()
        assert http(item_prefix+v3_identity)[0]==200
        v3.close();writer.close()
        return identity

    component={'type':'Quantity','name':'value','label':'Value',
               'definition':'https://example.org/value','uom':{'code':'1'}}
    ds=both_directions('datastream',prefix+'/datastreams:data',
                       '/systems/'+child+'/datastreams','/datastreams/',
                       {'name':'Live data '+token,'schema':{'obsFormat':'application/json','resultSchema':component}})
    cs=both_directions('controlstream',prefix+'/controlstreams:data',
                       '/systems/'+child+'/controlstreams','/controlstreams/',
                       {'uid':'urn:uuid:'+str(uuid.uuid4()),'name':'Live control '+token,
                        'inputName':'value','schema':{'commandFormat':'application/json','parametersSchema':component}})
    both_directions('observation',prefix+'/datastreams/'+ds+'/observations:data',
                    '/datastreams/'+ds+'/observations','/observations/',
                    {'resultTime':'2026-10-14T12:00:00Z','result':42})
    command=both_directions('command',prefix+'/controlstreams/'+cs+'/commands:data',
                            '/controlstreams/'+cs+'/commands','/commands/',{'parameters':21})
    cmd_path=prefix+'/controlstreams/'+cs+'/commands/'+command
    both_directions('commandstatus',cmd_path+'/status:data',
                    '/commands/'+command+'/status','/commands/'+command+'/status/',
                    {'reportTime':'2026-10-14T12:00:00Z','statusCode':'ACCEPTED'})
    both_directions('commandresult',cmd_path+'/result:data',
                    '/commands/'+command+'/result','/commands/'+command+'/result/',{'data':42})
    both_directions('systemevent',prefix+'/events:data',
                    '/systems/'+child+'/events','/systems/'+child+'/events/',
                    {'definition':'https://example.org/calibration','label':'Live event',
                     'time':'2026-10-14T12:00:00Z'})

    # Delete keeps operation tombstone and pre-delete ownership metadata.
    http('/systems/'+child+'?cascade=true', method='DELETE')
    _, event, _ = sub5.resource()
    assert event['type'] == 'org.ogc.api.csapi.system.delete'
    assert event['resourceparent'] == args.api+'/systems/'+root
    http('/systems/'+keyed_id, method='DELETE')
    sub5.resource()
    rejected('node1/systems:data', keyed, key='key-'+token, media='application/geo+json')
    sub5.quiet()
    sub5.close()

    if args.compose_project:
        def compose(*parts):
            subprocess.run(['docker','compose','-p',args.compose_project,*parts],check=True,stdout=subprocess.DEVNULL)
        # Accepted queued messages survive an API/broker process restart.
        session='persistent-'+token
        queue=client(client=session, clean=False)
        queue.subscribe('node1/systems:data');queue.close()
        queued_id=created(system('offline '+token))
        compose('restart','api')
        for attempt in range(60):
            try:
                http('/conformance');break
            except (OSError,urllib.error.URLError):
                time.sleep(.5)
        else:
            raise AssertionError('API did not recover')
        resumed=client(client=session,clean=False)
        assert resumed.session_present
        _,queued,_=resumed.resource();assert queued['id']==queued_id
        resumed.close()
        # Original key/bytes survive both restarts and changed client ID.
        c=client();assert c.subscribe('node1/systems:data')==2
        writer=client();assert writer.publish('node1/systems:data',q2body,key='qos2-'+token,media='application/geo+json')[0]==0x40
        _,replay,_=c.resource();assert replay['id']==q2id
        c.close();writer.close()
        def sql(statement):
            result = subprocess.run(['docker','compose','-p',args.compose_project,
                                     'exec','-T','db','psql','-U','postgres','-d',
                                     'connected_systems','-At','-v','ON_ERROR_STOP=1',
                                     '-c',statement],check=True,capture_output=True,text=True)
            return result.stdout.strip()

        def ready():
            for _ in range(60):
                try:
                    http('/conformance')
                    return
                except (OSError,urllib.error.URLError):
                    time.sleep(.5)
            raise AssertionError('API did not recover')

        # Kill the API while a database create is blocked before commit.
        # The create deadline is two seconds; the trigger deliberately
        # holds the database operation so the process can be killed in-flight.
        crash_key = 'crash-'+token
        ds_topic = 'node1/systems/'+root+'/datastreams:data'
        ds_body = json.dumps({'name':crash_key, 'schema':{'obsFormat':'application/json',
                               'resultSchema':{'type':'Quantity','name':'value','label':'Value','definition':'https://example.org/value',
                                               'uom':{'code':'1'}}}}).encode()
        sql("DROP TRIGGER IF EXISTS csapi_mqtt_test_sleep ON datastreams; CREATE OR REPLACE FUNCTION csapi_mqtt_test_sleep() RETURNS trigger LANGUAGE plpgsql AS $$ BEGIN PERFORM pg_sleep(8); RETURN NEW; END $$; "
            "CREATE TRIGGER csapi_mqtt_test_sleep BEFORE INSERT ON datastreams FOR EACH ROW EXECUTE FUNCTION csapi_mqtt_test_sleep()")
        def blocked_create():
            try:
                c = client()
                c.publish(ds_topic,ds_body,key=crash_key)
                c.close()
            except (OSError,EOFError):
                pass
        with concurrent.futures.ThreadPoolExecutor(max_workers=1) as pool:
            pending = pool.submit(blocked_create)
            for _ in range(40):
                if sql("SELECT count(*) FROM pg_stat_activity WHERE wait_event='PgSleep'") != '0':
                    break
                time.sleep(.05)
            else:
                sql('DROP TRIGGER csapi_mqtt_test_sleep ON datastreams; DROP FUNCTION csapi_mqtt_test_sleep()')
                raise AssertionError('create did not reach pre-commit crash point')
            compose('kill','-s','SIGKILL','api')
            pending.result(timeout=5)
        for _ in range(40):
            if sql("SELECT count(*) FROM pg_stat_activity WHERE wait_event='PgSleep'") == '0':
                break
            time.sleep(.25)
        sql("DROP TRIGGER csapi_mqtt_test_sleep ON datastreams; DROP FUNCTION csapi_mqtt_test_sleep()")
        assert sql("SELECT count(*) FROM mqtt_create_operations WHERE operation_key='"+crash_key+"'") == '0'
        assert sql("SELECT count(*) FROM datastreams WHERE name='"+crash_key+"'") == '0'
        compose('start','api')
        ready()
        watcher = client()
        assert watcher.subscribe(ds_topic) == 2
        c = client()
        ack = c.publish(ds_topic,ds_body,key=crash_key)
        assert ack[0]==0x40 and (len(ack[1])<3 or ack[1][2]<128), ack
        watcher.resource()
        c.close()
        assert sql("SELECT count(*) FROM mqtt_create_operations WHERE operation_key='"+crash_key+"'") == '1'
        assert sql("SELECT count(*) FROM datastreams WHERE name='"+crash_key+"'") == '1'
        watcher.close()

        # Lose the QoS1 acknowledgement after sending a create, then reconnect
        # with another client ID and retry the same key and input.
        lost_key='lost-ack-'+token
        lost_body=json.dumps({'name':lost_key,'schema':{'obsFormat':'application/json',
                              'resultSchema':{'type':'Quantity','name':'value','label':'Value','definition':'https://example.org/value','uom':{'code':'1'}}}}).encode()
        watcher=client();watcher.subscribe(ds_topic)
        c=client();c.publish(ds_topic,lost_body,key=lost_key,wait=False);c.close(abrupt=True)
        _,original,_=watcher.resource()
        c=client();assert c.publish(ds_topic,lost_body,key=lost_key)[0]==0x40
        _,replay,_=watcher.resource();assert original==replay
        assert sql("SELECT count(*) FROM datastreams WHERE name='"+lost_key+"'") == '1'
        c.close();watcher.close()

        # API outage closes the embedded MQTT endpoint; no automatic create retries.
        compose('stop','api')
        try:
            unavailable = client()
        except OSError:
            pass
        else:
            unavailable.close()
            raise AssertionError('MQTT listener survived API shutdown')
        compose('start','api')
    for identity in resources:
        try:
            http('/systems/'+identity+'?cascade=true',method='DELETE')
        except (OSError,urllib.error.URLError):
            pass
    print('PASS: MQTT 3.1.1/5 canonical ingress, hierarchy, fan-out, concurrent keys, QoS2 retransmission, packet reuse, ACLs, wills, retention'+(', restarts, offline queues, process crash, lost acknowledgement, outages' if args.compose_project else ''))

if __name__ == '__main__':
    main()
