import React from 'react';
import { Select, Button } from 'antd';
import axios from 'axios'
import CalendarSummary from './CalenderSummary'
const { Option } = Select;

class CalendarList extends React.Component {

  constructor(props) {

    super(props);

    this.baseLink = 'http://127.0.0.1:37555?format=json&code=' + props.code;
    this.state = {calendars: [], selections: [], data: null};
    axios.get(this.baseLink)
      .then( (response)  => {
        // handle success
        let options = response.data.map((item, index) => {
          return <Option key={index} value={item} label={item}>
            {item}
          </Option>
        })

        this.setState({calendars: options})
      })
  }

  getCalendar() {
    axios.get(this.baseLink + "&calendar=" + this.state.selections.join(",") )
      .then( (response)  => {
        // handle success

        console.log(response.data)
        this.setState({data: response.data})
      })
  }

  render() {

    return (
        <div>
          <Select
            mode="multiple"
            style={{ width: '100%' }}
            placeholder="select calenders"
            defaultValue={[]}
            onChange={
              (val, selections) => {
                this.setState({
                  selections: selections.map( (item) => {
                    return item.props.value
                  })
                })
              }
            }
            optionLabelProp="label"
          >
          {this.state.calendars}
        </Select>

        <Button onClick={() => {
          this.getCalendar()
        }}>
          Go
        </Button>

        <CalendarSummary data={this.state.data} />
        </div>
    );
  }
}

export default CalendarList;
