
#include <stdio.h>
#include <stdlib.h>

int foo() {
    printf("bye!\n");
    exit(0);
}

int main() {
    int i = 1;
    printf("hello world!\n");
    i += 2;
    printf("i is %d\n", i);
    foo();
}
