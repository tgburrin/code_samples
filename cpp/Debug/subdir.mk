################################################################################
# Automatically-generated file. Do not edit!
################################################################################

# Add inputs and outputs from these tool invocations to the build variables 
CPP_SRCS += \
../ApplicationException.cpp \
../MessageConsumer.cpp \
../PostgresCfg.cpp \
../PostgresDbh.cpp \
../ProcessCfg.cpp \
../SourceReference.cpp \
../kafka_client.cpp 

OBJS += \
./ApplicationException.o \
./MessageConsumer.o \
./PostgresCfg.o \
./PostgresDbh.o \
./ProcessCfg.o \
./SourceReference.o \
./kafka_client.o 

CPP_DEPS += \
./ApplicationException.d \
./MessageConsumer.d \
./PostgresCfg.d \
./PostgresDbh.d \
./ProcessCfg.d \
./SourceReference.d \
./kafka_client.d 


# Each subdirectory must supply rules for building sources it contributes
%.o: ../%.cpp
	@echo 'Building file: $<'
	@echo 'Invoking: GCC C++ Compiler'
	g++ -I/usr/include/postgresql -O0 -g3 -Wall -c -fmessage-length=0 -std=c++14 $(shell pkg-config --cflags libpq) $(shell pkg-config --cflags rdkafka++) -v -MMD -MP -MF"$(@:%.o=%.d)" -MT"$(@:%.o=%.d)" -o "$@" "$<"
	@echo 'Finished building: $<'
	@echo ' '


