
#include <stdio.h>
#include <stdlib.h>
#include <unistd.h>

#include "con_load.h"
#include "elevator_io_device.h"
#include "fsm.h"
#include "timer.h"
#include <pthread.h>
#include <stdbool.h>
#include <stdio.h>
#include <string.h>


// this struct needs a better name. It contains the hallbuttons, and a mutex
typedef struct  {
    bool HallButtons[N_FLOORS][2];
    pthread_mutex_t     mtx;
} HallButtonsWithMutex;


// this creates and initializes a hbwmtx
HallButtonsWithMutex * create_hb() {
    HallButtonsWithMutex * inputs = malloc(sizeof(HallButtonsWithMutex));

    for (int i = 0; i < N_FLOORS; i++)
    {
        inputs->HallButtons[i][0] = false;
        inputs->HallButtons[i][1] = false;
    }
    
    // init mutex
    pthread_mutex_init(&inputs->mtx, NULL);

    return inputs;
}

// reads inputs from the Go program, updates the hallbuttons struct 
void * readInputFromGo(void* args) {
    bool input[N_FLOORS][2];
    HallButtonsWithMutex* hallBtns = (HallButtonsWithMutex*)(args);
    while(1) {
        // poll the inputs
        
        // convert to inputs 

        pthread_mutex_lock(&hallBtns->mtx);

        for (int i = 0; i < N_FLOORS; i++)
        {
            hallBtns->HallButtons[i][0] = input[i][0];
            hallBtns->HallButtons[i][1] = input[i][1];
        }
        
        pthread_mutex_unlock(&hallBtns->mtx);
    }
}

int main(void) {

    // default code
    printf("Started!\n");
    
    int inputPollRate_ms = 25;
    con_load("elevator.con",
        con_val("inputPollRate_ms", &inputPollRate_ms, "%d")
    )
    
    ElevInputDevice input = elevio_getInputDevice();    
    
    if(input.floorSensor() == -1){
        fsm_onInitBetweenFloors();
    }
    
    // new code


    pthread_t pollGoInputThread;
    HallButtonsWithMutex * inputs = create_hb();

    pthread_create(&pollGoInputThread, NULL, readInputFromGo, inputs);
    

    while(1) {
        /*
        { // Request button
            static int prev[N_FLOORS][N_BUTTONS];
            for(int f = 0; f < N_FLOORS; f++){
                for(int b = 0; b < N_BUTTONS; b++){
                    int v = input.requestButton(f, b);
                    if(v  &&  v != prev[f][b]){
                        fsm_onRequestButtonPress(f, b);
                    }
                    prev[f][b] = v;
                }
            }
        }
        
        kode for å sende states ut
        

        les hallButtonsWithMutex
        

        */

        // ALL etterfølgende kode må være til stede, pluss input og output kode
        { // check cab calls
            static int prev[N_FLOORS][N_BUTTONS];
            for(int f = 0; f < N_FLOORS; f++) {
                int v = input.requestButton(f, 2);
                if(v  &&  v != prev[f][2]) {
                    fsm_onRequestButtonPress(f, 2);
                }
                prev[f][2] = v;
            }
        }
        
        

        { // Floor sensor
            static int prev = -1;
            int f = input.floorSensor();
            if(f != -1  &&  f != prev){
                fsm_onFloorArrival(f);
            }
            prev = f;
        }
        
        { // Timer
            if(timer_timedOut()){
                timer_stop();
                fsm_onDoorTimeout();
            }
        }
        
        usleep(inputPollRate_ms*1000);
    }
}









