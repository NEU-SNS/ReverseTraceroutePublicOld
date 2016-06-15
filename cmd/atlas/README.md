# Atlas

The Traceroute Atlas is used to find intersecting traceroutes toward destination which intersects at a hop. 

For Example:

The Reverse Traceroute system is working on a reverse traceroute from S back to D
The reverse traceroute consists of Hops D, H<sub>1</sub>, H<sub>2</sub>, H<sub>3</sub>.

In order to complete the reverse traceroute, a forward traceroute with a destination of S which intersects any of H<sub>1</sub>, H<sub>2</sub>, or H<sub>3</sub> can be inserted into the reverse traceroute.

A series of requests can be sent to the Atlas looking for a forward traceroute which has a destination of S and contains any of the addresses H<sub>1...3</sub>

If the Atlas has a traceroute which satisfies a request, the response will contain that traceroute. If there is no traceroue which satisfies the requests, the Atlas will run traceroutes to try to and satisfy the requests. The response to the requests will be a token which can be use to request any results from the traceroutes the Atlas ran, which satisfy the original requests.
	
