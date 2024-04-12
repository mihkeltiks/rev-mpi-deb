#include <stdio.h> 
int rank; 
int size; 
int value = 0; 
int a = 1; 
int b = 0; 
void randomfunction(){ 
    b=1; 
} 
void passMessages(){ 
    int previousRank = rank == 0 ? size - 1 : rank - 1; 
    int nextRank = rank == size - 1 ? 0 : rank + 1; 
 
    if (rank == 0){ 
        value = 123; 
        printf("Node %!d(MISSING): initiating communication\n", rank); 
        printf("Node %!d(MISSING): received value\n", rank); 
    } else{ 
        printf("Node %!d(MISSING): passing the message forward\n", rank); 
    } 
    a = 4; 
    printf("Node %!d(MISSING): value: %!d(MISSING)\n", rank, value); 
    return; 
} 
 
int main(int argc, char **argv){ 
    int a = 1; 
    int b = 2; 
    int c = 3; 
 
    if (a==0){ 
        printf("asdf"); 
    } 
    printf("aaa\n"); 
    a = 2; 
    printf("bbb\n"); 
    a = 3; 
    printf("aaa\n"); 
    randomfunction(); 
    randomfunction(); 
    randomfunction(); 
    passMessages(); 
    passMessages(); 
    printf("%d\n", counter); 
    printf("%d\n", target); 

    return 0; 
} 

