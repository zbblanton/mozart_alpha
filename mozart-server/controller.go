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
      //return false //This is unreachable until we fix the panic above.
  }

  //return true //This is unreachable
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
    newContainer.Worker = selectedWorker.AgentIP
    //fmt.Println("Worker:", worker.AgentIP)

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
  _, err = callSecuredAgent(serverTLSCert, serverTLSKey, caTLSCert, "POST", url, b)
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
  c.Worker = worker.AgentIP

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
  _, err = callSecuredAgent(serverTLSCert, serverTLSKey, caTLSCert, "POST", url, b)
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
  _, err = callSecuredAgent(serverTLSCert, serverTLSKey, caTLSCert, "GET", url, nil)
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
        //workers.Workers[worker.AgentIP] = worker
        //Save worker
        w1, err := json.Marshal(worker)
        if err != nil {
          panic(err)
        }
        ds.Put("mozart/workers/" + worker.AgentIP, w1)

        fmt.Println("Worker", worker.AgentIP, "has been set to disconnected.")

        //Get worker container run list
        var oldWorker Worker
        workerBytes, _ := ds.Get("mozart/workers/" + worker.AgentIP)
        if workerBytes != nil {
          err = json.Unmarshal(workerBytes, &oldWorker)
          if err != nil {
            panic(err)
          }
        }
        fmt.Println("The following container(s) will be moved:", oldWorker.Containers)

        //Move all containers on this worker
        for _, containerName := range oldWorker.Containers {
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
      if(checkWorkerHealth(worker.AgentIP, worker.AgentPort)){
        worker.Status = "connected"
        //Save worker
        w1, err := json.Marshal(worker)
        if err != nil {
          panic(err)
        }
        ds.Put("mozart/workers/" + worker.AgentIP, w1)
        //workers.Workers[worker.AgentIP] = worker
        fmt.Println("Worker", worker.AgentIP, "has been set to connected.")
        return true
      }
      return false
    default:
      panic("Not action available for Worker Controller.")
      //return false //This is unreachable until we fix the panic above.
  }

  //return true //This is unreachable
}
