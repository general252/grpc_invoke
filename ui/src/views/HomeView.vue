<template>
  <el-select v-model="serviceValue" placeholder="服务" @change="serviceChange" filterable="true">
    <el-option v-for="item in services" :key="item.service_name" :label="item.service_name"
      :value="item.service_name" />
  </el-select>

  <el-select v-model="methodValue" placeholder="方法" @change="methodChange" filterable="true">
    <el-option v-for="item in methods" :key="item.method_name" :label="item.method_name" :value="item.method_name" />
  </el-select>

  <el-button type="primary" @click="invoke">提交</el-button>
  <el-button type="primary" @click="cleanResponse">清空</el-button>
  <br />
  <br />


  <el-tabs type="border-card" v-model="tabSelect">
    <el-tab-pane label="请求" name="tabItemReq">

      <!-- json-editer -->
      <div id="editor_request" style="background-color: rgba(250, 250, 250, 0.5)"></div>

    </el-tab-pane>

    <el-tab-pane label="请求Header" name="tabItemReqHead">

      <!-- json-editer -->
      <div id="editor_request_header" style="background-color: rgba(250, 250, 250, 0.5)"></div>

    </el-tab-pane>

    <el-tab-pane label="回复" name="tabItemRes">

      <!-- json-viewer -->
      <json-viewer :value="jsonData" :expand-depth="5" copyable boxed sort expanded="true"></json-viewer>

    </el-tab-pane>

    <el-tab-pane label="回复Header" name="tabItemResHeader">

      <!-- json-viewer -->
      <json-viewer :value="jsonRpcData" :expand-depth="5" copyable boxed sort expanded="true"></json-viewer>

    </el-tab-pane>
  </el-tabs>

</template>

<script>
import axios from "axios";
import { ElNotification } from "element-plus";

export default {
  data() {
    return {
      serviceValue: "",
      services: [
        {
          service_name: "serv",
          methods: [
            {
              //method_name: "getName",
            },
          ],
        },
      ],

      methodValue: "",
      methods: [
        {
          // method_name: "getA",
        },
      ],

      tabSelect: "tabItemReq",

      jsonEditorRequest: null,
      jsonEditorRequestHeader: null,
      jsonRpcData: {},
      jsonData: {},
    };
  },

  methods: {
    getServiceList() {
      let pThis = this;
      axios
        .get("/rpc/services", {})
        .then(function (response) {
          pThis.services = response.data;
        })
        .catch(function (error) {
          console.log(error);
        })
        .then(function () {
          // 总是会执行
        });
    },

    serviceChange(val) {
      console.log(val);

      this.services.forEach((element) => {
        if (element.service_name == val) {
          this.methods = element.methods;
        }
      });
    },
    methodChange(val) {
      console.log(this.serviceValue, this.methodValue);
      if (
        typeof this.serviceValue == "undefined" ||
        typeof this.methodValue == "undefined"
      ) {
        return;
      }

      if (this.serviceValue.length > 0 && this.methodValue.length > 0) {
        let pThis = this;

        axios
          .get(`/rpc/jsonSchema/${this.serviceValue}/${this.methodValue}`, {})
          .then(function (response) {
            console.log(response);

            console.log(response.data);

            if (pThis.jsonEditorRequest) {
              pThis.jsonEditorRequest.destroy();
              pThis.jsonEditorRequest = null;
            }
            // Initialize the editor
            let dom = document.getElementById("editor_request")
            pThis.jsonEditorRequest = new window.JSONEditor(
              dom,
              {
                schema: response.data,
                max_depth: 0,
                compact: true,
              }
            );

            pThis.tabSelect = "tabItemReq"
          })
          .catch(function (error) {
            console.log(error);
            ElNotification({
              title: "Warning",
              message: JSON.stringify(error.response.data.error, null, 2),
              type: "warning",
              position: "bottom-right",
            });
          })
          .then(function () {
            // 总是会执行
          });
      }
    },

    cleanResponse() {
      this.jsonData = '{}'
    },
    invoke() {
      if (typeof this.jsonEditorRequest == "undefined") {
        return;
      }

      var validation_errors = this.jsonEditorRequest.validate();
      // Show validation errors if there are any
      if (validation_errors.length) {
        ElNotification({
          title: "Warning",
          message: JSON.stringify(validation_errors, null, 2),
          type: "warning",
          position: "bottom-right",
        });
        return;
      }

      let payload = this.jsonEditorRequest.getValue();
      let header = this.jsonEditorRequestHeader.getValue();
      // let value = JSON.stringify(payload, null, 2);

      let headerMap = new Map()
      header.forEach(element => {
        headerMap[element.key] = element.value
      });

      /////////////////////////////////////////////////////////////////////////////////////////
      if (
        typeof this.serviceValue == "undefined" ||
        typeof this.methodValue == "undefined"
      ) {
        return;
      }

      if (this.serviceValue.length > 0 && this.methodValue.length > 0) {
        let pThis = this;

        axios
          .post(`/rpc/invoke/${this.serviceValue}/${this.methodValue}`, {
            header: headerMap,
            data: payload,
          })
          .then(function (response) {
            // console.log(response.data);
            let result = response.data;

            pThis.tabSelect = "tabItemRes"

            pThis.jsonRpcData = {
              header: result.header,
              trailer: result.trailer,
            }
            pThis.jsonData = result.data;
          })
          .catch(function (error) {
            console.log(error);
            ElNotification({
              title: "Warning",
              message: JSON.stringify(error.response.data.error, null, 2),
              type: "warning",
              position: "bottom-right",
            });
          })
          .then(function () {
            // 总是会执行
          });
      }
    },
  },

  mounted() {
    let jsonSchema = JSON.parse(`{
        "type": "array",
        "title": "header",
        "items": {
          "type": "object",
          "title": "key/value",
          "properties": {
            "key": {
              "type": "string"
            },
            "value": {
              "type": "string"
            }
          }
        }
    }`);
    this.jsonEditorRequestHeader = new window.JSONEditor(
      document.getElementById("editor_request_header"),
      {
        schema: jsonSchema,
        compact: true,
      }
    );

    this.getServiceList();
  },
};
</script>


<style>
.el-tabs__content {
  min-height: 700px;
  ;
}
</style>