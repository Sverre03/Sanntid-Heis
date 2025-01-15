// Compile with `gcc foo.c -Wall -std=gnu99 -lpthread`, or use the makefile
// The executable will be named `foo` if you use the makefile, or `a.out` if you use gcc directly

#include <pthread.h>
#include <stdio.h>

int i = 0;
pthread_mutex_t sharedVariableMutex;

// Note the return type: void*
void* incrementingThreadFunction(){
    // TODO: increment i 1_000_000 times
    
    for (int j = 0; j < 1000000; j++)
    {
        pthread_mutex_lock(&sharedVariableMutex);
        i++;
        pthread_mutex_unlock(&sharedVariableMutex);
    }
    
    
    return NULL;
}

void* decrementingThreadFunction(){
    // TODO: decrement i 1_000_000 times
    for (int j = 0; j < 1000000; j++)
    {
        pthread_mutex_lock(&sharedVariableMutex);
        i--;
        pthread_mutex_unlock(&sharedVariableMutex);
    }
    
    
    return NULL;
}


int main(){
    pthread_t incrementThread;
    pthread_t decrementThread;
    // TODO: 
    // start the two functions as their own threads using `pthread_create`
    // Hint: search the web! Maybe try "pthread_create example"?
    pthread_create(&incrementThread, NULL, incrementingThreadFunction, NULL);
    pthread_create(&decrementThread, NULL, decrementingThreadFunction, NULL);

    // TODO:
    // wait for the two threads to be done before printing the final result
    // Hint: Use `pthread_join`    
    pthread_join(incrementThread, NULL);
    pthread_join(decrementThread, NULL);
    
    printf("The magic number is: %d\n", i);
    return 0;
}
