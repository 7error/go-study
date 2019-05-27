<!--
 Copyright 2019 Yandy Ramirez
 
 Licensed under the Apache License, Version 2.0 (the "License");
 you may not use this file except in compliance with the License.
 You may obtain a copy of the License at
 
     http://www.apache.org/licenses/LICENSE-2.0
 
 Unless required by applicable law or agreed to in writing, software
 distributed under the License is distributed on an "AS IS" BASIS,
 WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 See the License for the specific language governing permissions and
 limitations under the License.
-->

## The Coffman Conditions for Deadlocks

Deadlocks also **suck**

-   Mutual Exclusion
    -   A concurrent process holds exclusive rights to a resource at any one time
-   Wait For Condition
    -   A concurrent process must simultaneously hold a resource and be waiting for an additional resource
-   No Preemption
    -   A resource held by a concurrent process can only be released by that process, so it fulfills this condition
-   Circular Wait
    -   A concurrent process (P1) must be waiting on a chain of other concurrent processes (P2), which are in turn waiting on it (P1), so it fulfills this final condition

## Race Conditions

They **suck**, so avoid them, just because a line appears before another line of code does not guarantee execution order.

## Lifelocks

No not the identity service, when two or more actions are being performed but these actions do nothing to move the state of the program forward.

## Starvation

When I'm hungry, which is all the time... no but really.

Similar to lifelocks, except in a lifelock situation all processes are starved equally. With `starvation` only one or a few of all processes cannot get the resources needed to perform the operations.