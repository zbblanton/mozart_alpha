package main

import(
  //"os"
  "fmt"
  "time"
  "bytes"
  "encoding/json"
  //"net/http"
)

//////////////////////////////////////////////////////////
func containerControllerQueue(messages chan interface{}) {
  //Set a ticker for a small delay (may not be needed for this queue)
  //Range through the messages, running executor on each
  //if it fails, add the retry queue
  ticker := time.NewTicker(time.Second)
  for message := range messages {
      //message := message.(test)
      //fmt.Println("Message", message.test2, time.Now())
      if(!containerControllerExecutor(message)){
        containerRetryQueue <- message
      }
      <- ticker.C
  }
}

func containerControllerRetryQueue(messages chan interface{}) {
  //Set a ticker for a retry delay (careful, make sure the delay is what you want)
  //Range through the messages, running executor on each
  //if it fails, add to the retry queue again
  ticker := time.NewTicker(5 * time.Second)
  for message := range messages {
      if(!containerControllerExecutor(message)){
        containerRetryQueue <- message
      }
      <- ticker.C
  }
}

func containerControllerExecutor(msg interface{}) bool{
  //Case for each command, run the function matching the command and struct type
  switch msg.(type) {
    case ContainerConfig:
      msg := msg.(ContainerConfig)
      return containerControllerStart(msg)
    case Container:
      msg := msg.(Container)
      return containerControllerMove(msg)
    case string:
      msg := msg.(string)
      return containerControllerStop(msg)
    default:
      panic("Not action available for Container Controller.")
      return false
  }

  return true
}

func containerControllerStart(c ContainerConfig) bool {
  newContainer := Container{
    Name: c.Name,
    State: "starting",
    DesiredState: "running",
    Config: c}

    //Save container
    c1, err := json.Marshal(newContainer)
    if err != nil {
      panic(err)
    }
    ds.Put("mozart/containers/" + newContainer.Name, c1)
    //containers.mux.Lock()
    ////config.Containers = append(config.Containers, newContainer)
    //containers.Containers[c.Name] = newContainer
    //writeFile("containers", "containers.data")
    //containers.mux.Unlock()

    selectedWorker, err := selectWorker()
    if err != nil {
      fmt.Println("Error:", err)
      return false
    }
    newContainer.Worker = selectedWorker.AgentIp
    //fmt.Println("Worker:", worker.AgentIp)

    //Save container
    c1, err = json.Marshal(newContainer)
    if err != nil {
      panic(err)
    }
    ds.Put("mozart/containers/" + newContainer.Name, c1)

    //Update workers container run list
    var worker Worker
    workerBytes, _ := ds.Get("mozart/workers/" + newContainer.Worker)
    err = json.Unmarshal(workerBytes, &worker)
    if err != nil {
      panic(err)
    }
    worker.Containers[newContainer.Name] = newContainer.Name
    workerToBytes, err := json.Marshal(worker)
    if err != nil {
      panic(err)
    }
    ds.Put("mozart/workers/" + newContainer.Worker, workerToBytes)
    //containers.mux.Lock()
    ////config.Containers = append(config.Containers, newContainer)
    //containers.Containers[c.Name] = newContainer
    //writeFile("containers", "containers.data")
    //containers.mux.Unlock()

  //Will need to add support for the worker key!!!!!
  type CreateReq struct {
    Key string
    Container Container
  }
  j := CreateReq{Key: "NEEDTOADDSUPPORTFORTHIS!!!", Container: newContainer}
  b := new(bytes.Buffer)
  json.NewEncoder(b).Encode(j)
  url := "https://" + newContainer.Worker + ":49433" + "/create"
  _, err = callSecuredAgent(serverTlsCert, serverTlsKey, caTlsCert, "POST", url, b)
  if err != nil {
		//panic(err)
    return false
	}

  return true
}

func containerControllerMove(c Container) bool {
  //Remove container from workers container run list
  var oldWorker Worker
  workerBytes, _ := ds.Get("mozart/workers/" + c.Worker)
  err := json.Unmarshal(workerBytes, &oldWorker)
  if err != nil {
    panic(err)
  }
  delete(oldWorker.Containers, c.Name)
  workerToBytes, err := json.Marshal(oldWorker)
  if err != nil {
    panic(err)
  }
  ds.Put("mozart/workers/" + c.Worker, workerToBytes)

  //Clear worker
  c.State = "moving"
  c.Worker = ""

  //Save container
  c1, err := json.Marshal(c)
  if err != nil {
    panic(err)
  }
  ds.Put("mozart/containers/" + c.Name, c1)
  // containers.mux.Lock()
  // //config.Containers = append(config.Containers, newContainer)
  // containers.Containers[c.Name] = c
  // writeFile("containers", "containers.data")
  // containers.mux.Unlock()

  worker, err := selectWorker()
  if err != nil {
    fmt.Println("Error:", err)
    return false
  }
  c.Worker = worker.AgentIp

  //Save container
  c1, err = json.Marshal(c)
  if err != nil {
    panic(err)
  }
  ds.Put("mozart/containers/" + c.Name, c1)

  //Update workers container run list
  //var worker Worker
  workerBytes, _ = ds.Get("mozart/workers/" + c.Worker)
  err = json.Unmarshal(workerBytes, &worker)
  if err != nil {
    panic(err)
  }
  worker.Containers[c.Name] = c.Name
  workerToBytes, err = json.Marshal(worker)
  if err != nil {
    panic(err)
  }
  ds.Put("mozart/workers/" + c.Worker, workerToBytes)
  // containers.mux.Lock()
  // //config.Containers = append(config.Containers, newContainer)
  // containers.Containers[c.Name] = c
  // writeFile("containers", "containers.data")
  // containers.mux.Unlock()

  //Will need to add support for the worker key!!!!!
  type CreateReq struct {
    Key string
    Container Container
  }
  j := CreateReq{Key: "NEEDTOADDSUPPORTFORTHIS!!!", Container: c}
  b := new(bytes.Buffer)
  json.NewEncoder(b).Encode(j)
  url := "https://" + c.Worker + ":49433" + "/create"
  _, err = callSecuredAgent(serverTlsCert, serverTlsKey, caTlsCert, "POST", url, b)
  if err != nil {
		//panic(err)
    return false
	}

  return true
}

func containerControllerStop(name string) bool {
  //Update container desired state
  // containers.mux.Lock()
  // container := containers.Containers[name]
  // container.DesiredState = "stopped"
  // containers.Containers[name] = container
  // writeFile("containers", "containers.data")
  // containers.mux.Unlock()
  //Get container
  var container Container
  c, _ := ds.Get("mozart/containers/" + name)
  err := json.Unmarshal(c, &container)
  if err != nil {
    panic(err)
  }
  //Change desired state
  container.DesiredState = "stopped"
  //Save new desired state
  b2, err := json.Marshal(container)
  if err != nil {
    panic(err)
  }
  ds.Put("mozart/containers/" + name, b2)



  //Will need to add support for the worker key!!!!!
  url := "https://" + container.Worker + ":49433" + "/stop/" + container.Name
  _, err = callSecuredAgent(serverTlsCert, serverTlsKey, caTlsCert, "GET", url, nil)
  if err != nil {
		//panic(err)
    return false
	}

  return true
}

//////////////////////////////////////////////////////////





func workerControllerQueue(messages chan ControllerMsg) {
  //Set a ticker for a small delay (may not be needed for this queue)
  //Range through the messages, running executor on each
  //if it fails, add the retry queue
  ticker := time.NewTicker(time.Second)
  for message := range messages {
      //message := message.(test)
      //fmt.Println("Message", message.test2, time.Now())
      if(!workerControllerExecutor(message)){
        workerRetryQueue <- message
      }
      <- ticker.C
  }
}

func workerControllerRetryQueue(messages chan ControllerMsg) {
  //Set a ticker for a retry delay (careful, make sure the delay is what you want)
  //Range through the messages, running executor on each
  //if it fails, add to the retry queue again
  ticker := time.NewTicker(5 * time.Second)
  for message := range messages {
      if(!workerControllerExecutor(message)){
        workerRetryQueue <- message
      }
      <- ticker.C
  }
}

func workerControllerExecutor(msg ControllerMsg) bool{
  //Case for each command, run the function matching the command and struct type
  fmt.Println("Controller executing action:", msg.Action)
  switch msg.Action {
    case "reconnect":
      worker := msg.Data.(ControllerReconnectMsg).worker
      currentTime := time.Now()
      //disconnectTime := msg.Data.timesomething.Add(time.Minute)
      disconnectTime := msg.Data.(ControllerReconnectMsg).disconnectTime
      if(currentTime.Sub(disconnectTime).Seconds() >= 60){
        worker.Status = "disconnected"
        //workers.Workers[worker.AgentIp] = worker
        //Save worker
        w1, err := json.Marshal(worker)
        if err != nil {
          panic(err)
        }
        ds.Put("mozart/workers/" + worker.AgentIp, w1)

        fmt.Println("Worker", worker.AgentIp, "has been set to disconnected.")

        //Get worker container run list
        var worker Worker
        workerBytes, _ := ds.Get("mozart/workers/" + worker.AgentIp)
        if workerBytes != nil {
          err = json.Unmarshal(workerBytes, &worker)
          if err != nil {
            panic(err)
          }
        }

        //Move all containers on this worker
        for _, containerName := range worker.Containers {
          var container Container
          c, _ := ds.Get("mozart/containers/" + containerName)
          err = json.Unmarshal(c, &container)
          if err != nil {
            panic(err)
          }
          containerQueue <- container
        }
        return true
      }
      if(checkWorkerHealth(worker.AgentIp, worker.AgentPort)){
        worker.Status = "connected"
        //Save worker
        w1, err := json.Marshal(worker)
        if err != nil {
          panic(err)
        }
        ds.Put("mozart/workers/" + worker.AgentIp, w1)
        //workers.Workers[worker.AgentIp] = worker
        fmt.Println("Worker", worker.AgentIp, "has been set to connected.")
        return true
      } else {
        return false
      }
    default:
      panic("Not action available for Worker Controller.")
      return false
  }

  return true
}











/*
func controllerContainersStart(c Container){
  //Will need to add support for the worker key!!!!!
  type CreateReq struct {
    Key string
    Container ContainerConfig
  }

  j := CreateReq{Key: "NEEDTOADDSUPPORTFORTHIS!!!", Container: c.Config}

  b := new(bytes.Buffer)
  json.NewEncoder(b).Encode(j)
  url := "https://" + c.Worker + ":49433" + "/create"

  _, err := callSecuredAgent(serverTlsCert, serverTlsKey, caTlsCert, "POST", url, b)
  if err != nil {
		panic(err)
	}
}

func controllerContainersStop(c Container){
  //Will need to add support for the worker key!!!!!
  type CreateReq struct {
    Key string
    Container ContainerConfig
  }

  url := "https://" + c.Worker + ":49433" + "/stop/" + c.Name

  _, err := callSecuredAgent(serverTlsCert, serverTlsKey, caTlsCert, "GET", url, nil)
  if err != nil {
		panic(err)
	}
}

func controllerContainers() {
  //TODO: We need to add an initializing part so that we can get get
  //containers statuses before we start looping.
  for {
    //Loop through containers and make sure the desiredState matches the state, if not, perform DesiredState action.
    containers.mux.Lock()
    for key, container := range containers.Containers {
      if(container.State != container.DesiredState){
        if(container.DesiredState == "running"){
          //Run function to start a container
          //Below we assume that the containers actually start and put in a running state. Will need to add actual checks.
          controllerContainersStart(container)
          container.State = "running"
          containers.Containers[key] = container
          writeFile("containers", "containers.data")
          fmt.Print(container)
        } else if(container.DesiredState == "stopped"){
          //Run function to start a container
          //Below we assume that the containers actually start and put in a running state. Will need to add actual checks.
          controllerContainersStop(container)
          container.State = "stopped"
          containers.Containers[key] = container
          writeFile("containers", "containers.data")
          fmt.Print(container)
        }
      }
    }
    containers.mux.Unlock()
    fmt.Println("Waiting 15 seconds!")
    time.Sleep(time.Duration(15) * time.Second)
  }
  os.Exit(1) //In case the for loop exits, stop the whole program.
}
*/
