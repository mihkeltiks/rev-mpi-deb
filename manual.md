This document contains detailled instructions for using the different backends. It also lists potential problems and solutions.

CRIU
----

We have not tested the possibilities to run CRIU in user space. root privileges were used. The tested CRIU version was 3.19.

DMTCP
-----

We have tested this on Ubuntu  20.04.6 LTS. DMTCP 3.2.0 was installed from the github repository at https://github.com/dmtcp/dmtcp/tree/main. MPI was mpich, installed via apt-get, version 3.3.2.

When starting the debugger, there is a warning "Application trying to use DMTCP's signal for it's own use". This is due to golang using the SIGUSR signals for its own purposes. The warning can be ignored since the signals are still received by DMTCP as required.

If during a restore attempt, there is an error "Message: Failed to restore this process as session leader" (usually printed in red), then there is a problem with privileges. Running the debugger as root solves that problem. A better solution would be welcome.