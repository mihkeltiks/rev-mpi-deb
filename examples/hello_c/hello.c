
#include <stdio.h>
#include <stdlib.h>
#include <limits.h>

int global = 1234;
int max = INT_MAX;
int min = INT_MIN;


int foo() {
    printf("bye!\n");
    exit(0);
}

int main() {
    int i = 1;
    printf("hello world!\n");
    i += 2;
    printf("i is %d\n", i);
    global = 4567;
    foo();
}
