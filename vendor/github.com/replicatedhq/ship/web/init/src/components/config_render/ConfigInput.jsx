import React from "react";
import ConfigItemTitle from "./ConfigItemTitle";

export default class ConfigInput extends React.Component {

  constructor(props) {
    super(props)
    this.inputRef = React.createRef();
    this.state = {
      inputVal: "",
      focused: false
    }
  }

  handleOnChange = (e) => {
    const { handleOnChange, name } = this.props;
    this.setState({ inputVal: e.target.value });
    if (handleOnChange && typeof handleOnChange === "function") {
      handleOnChange(name, e.target.value);
    }
  }

  componentDidUpdate(lastProps) {
    if (this.props.value !== lastProps.value && !this.state.focused) {
      this.setState({ inputVal: this.props.value });
    }
  }

  componentDidMount() {
    if (this.props.value) {
      this.setState({ inputVal: this.props.value });
    }
  }

  render() {
    return (
      <div className={`field field-type-text ${this.props.hidden ? "hidden" : "u-marginTop--15"}`}>
        {this.props.title !== "" ?
          <ConfigItemTitle
            title={this.props.title}
            recommended={this.props.recommended}
            required={this.props.required}
            name={this.props.name}
          />
          : null}
        {this.props.help_text !== "" ? <p className="field-section-help-text u-marginTop--small u-lineHeight--normal">{this.props.help_text}</p> : null}
        <div className="field-input-wrapper u-marginTop--15">
          <input
            ref={this.inputRef}
            type={this.props.inputType}
            {...this.props.props}
            placeholder={this.props.default}
            value={this.state.inputVal}
            readOnly={this.props.readonly}
            disabled={this.props.readonly}
            onChange={(e) => this.handleOnChange(e)}
            onFocus={() => this.setState({ focused: true })}
            onBlur={() => this.setState({ focused: false })}
            className={`${this.props.className || ""} Input ${this.props.readonly ? "readonly" : ""}`} />
        </div>
      </div>
    );
  }
}
