#!/usr/bin/python

import sys,xmlrpclib,socket, sys, getopt
from sqlalchemy import Column
from sqlalchemy.ext.declarative import declarative_base
from sqlalchemy import create_engine
from sqlalchemy.dialects.mysql import INTEGER, TINYINT, VARCHAR, DATETIME 
from sqlalchemy.orm import sessionmaker
from sqlalchemy import orm
from sqlalchemy import func
import struct

conn_string = 'mysql://{0}:{1}@{2}:{3}/plcontroller' 

def ip2int(addr):                                                               
    return struct.unpack("!I", socket.inet_aton(addr))[0]


Base = declarative_base()

class VantagePoint(Base):
    __tablename__ = 'vantage_point'
    
    ip = Column(INTEGER(unsigned=True), primary_key=True)
    controller = Column(INTEGER(unsigned=True), nullable=True)
    hostname = Column(VARCHAR(length=255), default='')
    site = Column(VARCHAR(length=255), default='')
    timestamp = Column(TINYINT(), default=0)
    record_route = Column(TINYINT(), default=0)
    can_spoof = Column(TINYINT(), default=0)
    receive_spoof = Column(TINYINT(), default=0)
    port = Column(INTEGER(), default=0)
    last_health_check = Column(VARCHAR(length=255), default='')
    last_updated = Column(DATETIME(), default=func.current_timestamp())
    spoof_checked = Column(DATETIME(), nullable=True)

    def __repr__(self):
        return "<VantagePoint(ip='%d', hostname='%s', site='%s')>" % (
                self.ip, self.hostname, self.site)





def resolveName(hostname):
    try:
        ip  = socket.gethostbyname(hostname)
        return ip
    except socket.error:
        return None

usage = "Usage: plnodes.py -u username -p password -a address -o port"

def main(argv):
    uname = ''
    password = ''
    addr = ''
    port = ''
    try:
        opts, args = getopt.getopt(argv, "hu:p:a:o:")
    except getopt.GetoptError:
        print usage
        sys.exit(1)
    
    for opt, arg in opts:
        if opt == '-h':
            print usage
            sys.exit()
        if opt == '-u':
            uname = arg
        elif opt == '-p':
            password = arg
        elif opt == '-a':
            addr = arg
        elif opt == '-o':
            port = arg
    if uname == '' or password == '' or addr == '' or port == '':
        print usage
        sys.exit(1)

    SLICE_ID = 22129
    api_server = xmlrpclib.ServerProxy('https://www.planet-lab.org/PLCAPI/', allow_none=True)

    auth = { }

    auth['AuthMethod'] = 'password'

    auth['Username'] = 'rhansen2@ccs.neu.edu'
    auth['AuthString'] = 'Allislost1.'

    authorized = api_server.AuthCheck(auth)

    in_slice = []
    nin_slice = []
    add_to_slice = []
    node_ids = []
    nodeid_to_props = {}

    if authorized:
        nodes = api_server.GetNodes(auth)
        for x in nodes:
            node_ids.append(x['node_id'])
            x['ip_addr'] = resolveName(x['hostname'])
            nodeid_to_props[x['node_id']] = x

            if SLICE_ID in x['slice_ids']:
                in_slice.append(x)
            else:
                nin_slice.append(x)
                add_to_slice.append(x['hostname'])


        attrs = api_server.GetSliceTags(auth, {'node_id': node_ids, 'tagname': 'ip_addresses'})
        for attr in attrs:
            nid = attr['node_id']
            ips = attr['value']
            ip = ips.split(",")
            nodeid_to_props[nid]['ip_addr'] = ip[0]

         
        engine = create_engine(conn_string.format(uname, password, addr, port))
        Session = sessionmaker()
        Session.configure(bind=engine)
        session = Session()
        for key in nodeid_to_props.keys():
            curr = nodeid_to_props[key]
            hostname = curr['hostname']
            ip_addr = curr['ip_addr']
            if "mlab" in hostname:
                port = 806
            else:
                port = 22
            try:
                vp = session.query(VantagePoint).filter(VantagePoint.hostname==hostname).one()
                if ip_addr is not None:
                    vp.ip_addr = ip_addr
                    session.add(vp)
            except orm.exc.NoResultFound:
                if ip_addr is not None:
                    nvp = VantagePoint(ip = ip2int(ip_addr), hostname=hostname, port=port)
                    session.add(nvp)
        session.commit()
        api_server.AddSliceToNodes(auth, SLICE_ID, add_to_slice)

if __name__ == "__main__":
    main(sys.argv[1:])
