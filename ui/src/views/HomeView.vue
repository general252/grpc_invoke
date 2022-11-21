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
      <div id="editor_holder" style="background-color: rgba(250, 250, 250, 0.5)"></div>

    </el-tab-pane>
    <el-tab-pane label="回复" name="tabItemRes">

      <!-- json-viewer -->
      <json-viewer :value="jsonData" :expand-depth="5" copyable boxed sort expanded="true"></json-viewer>

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

      jsonEditor: null,
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

      if (this.serviceChange.length > 0 && this.methodValue.length > 0) {
        let pThis = this;

        axios
          .get(`/rpc/jsonSchema/${this.serviceValue}/${this.methodValue}`, {})
          .then(function (response) {
            console.log(response);

            console.log(response.data);

            if (pThis.jsonEditor) {
              pThis.jsonEditor.destroy();
              pThis.jsonEditor = null;
            }
            // Initialize the editor
            let dom = document.getElementById("editor_holder")
            pThis.jsonEditor = new window.JSONEditor(
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
      if (typeof this.jsonEditor == "undefined") {
        return;
      }

      var validation_errors = this.jsonEditor.validate();
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

      let json = this.jsonEditor.getValue();

      let value = JSON.stringify(json, null, 2);

      /////////////////////////////////////////////////////////////////////////////////////////
      if (
        typeof this.serviceValue == "undefined" ||
        typeof this.methodValue == "undefined"
      ) {
        return;
      }

      if (this.serviceChange.length > 0 && this.methodValue.length > 0) {
        let pThis = this;

        axios
          .post(`/rpc/invoke/${this.serviceValue}/${this.methodValue}`, json)
          .then(function (response) {
            // console.log(response.data);

            pThis.tabSelect = "tabItemRes"

            pThis.jsonData = response.data;
          })
          .catch(function (error) {
            console.log(error);
            ElNotification({
              title: "Warning",
              message: JSON.stringify(error.response.data, null, 2),
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