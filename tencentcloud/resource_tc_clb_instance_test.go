package tencentcloud

import (
	"context"
	"fmt"
	"testing"

	"github.com/hashicorp/terraform-plugin-sdk/helper/resource"
	"github.com/hashicorp/terraform-plugin-sdk/terraform"
)

func TestAccTencentCloudClbInstance_basic(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckClbInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccClbInstance_basic,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClbInstanceExists("tencentcloud_clb_instance.clb_basic"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.clb_basic", "network_type", "OPEN"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.clb_basic", "clb_name", "tf-clb-basic"),
				),
			},
			{
				ResourceName:      "tencentcloud_clb_instance.clb_basic",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccTencentCloudClbInstance_open(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckClbInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccClbInstance_open,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClbInstanceExists("tencentcloud_clb_instance.clb_open"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.clb_open", "network_type", "OPEN"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.clb_open", "clb_name", "tf-clb-open"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.clb_open", "project_id", "0"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.clb_open", "security_groups.#", "1"),
					resource.TestCheckResourceAttrSet("tencentcloud_clb_instance.clb_open", "security_groups.0"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.clb_open", "target_region_info_region", "ap-guangzhou"),
					resource.TestCheckResourceAttrSet("tencentcloud_clb_instance.clb_open", "target_region_info_vpc_id"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.clb_open", "tags.test", "tf"),
				),
			},
			{
				Config: testAccClbInstance_update_open,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClbInstanceExists("tencentcloud_clb_instance.clb_open"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.clb_open", "clb_name", "tf-clb-update-open"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.clb_open", "network_type", "OPEN"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.clb_open", "project_id", "0"),
					resource.TestCheckResourceAttrSet("tencentcloud_clb_instance.clb_open", "vpc_id"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.clb_open", "security_groups.#", "1"),
					resource.TestCheckResourceAttrSet("tencentcloud_clb_instance.clb_open", "security_groups.0"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.clb_open", "target_region_info_region", "ap-guangzhou"),
					resource.TestCheckResourceAttrSet("tencentcloud_clb_instance.clb_open", "target_region_info_vpc_id"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.clb_open", "tags.test", "test"),
				),
			},
		},
	})
}

func TestAccTencentCloudClbInstance_internal(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckClbInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccClbInstance_internal,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClbInstanceExists("tencentcloud_clb_instance.clb_internal"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.clb_internal", "clb_name", "tf-clb-internal"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.clb_internal", "network_type", "INTERNAL"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.clb_internal", "project_id", "0"),
					resource.TestCheckResourceAttrSet("tencentcloud_clb_instance.clb_internal", "vpc_id"),
					resource.TestCheckResourceAttrSet("tencentcloud_clb_instance.clb_internal", "subnet_id"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.clb_internal", "tags.test", "tf1"),
				),
			},
			{
				Config: testAccClbInstance_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClbInstanceExists("tencentcloud_clb_instance.clb_internal"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.clb_internal", "clb_name", "tf-clb-update-internal"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.clb_internal", "network_type", "INTERNAL"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.clb_internal", "project_id", "0"),
					resource.TestCheckResourceAttrSet("tencentcloud_clb_instance.clb_internal", "vpc_id"),
					resource.TestCheckResourceAttrSet("tencentcloud_clb_instance.clb_internal", "subnet_id"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.clb_internal", "tags.test", "test"),
				),
			},
			{
				ResourceName:      "tencentcloud_clb_instance.clb_internal",
				ImportState:       true,
				ImportStateVerify: true,
			},
		},
	})
}

func TestAccTencentCloudClbInstance_default_enable(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckClbInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccClbInstance_default_enable,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClbInstanceExists("tencentcloud_clb_instance.default_enable"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.default_enable", "network_type", "OPEN"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.default_enable", "clb_name", "my_open_clb"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.default_enable", "project_id", "0"),
					resource.TestCheckResourceAttrSet("tencentcloud_clb_instance.default_enable", "vpc_id"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.default_enable", "load_balancer_pass_to_target", "true"),
					resource.TestCheckResourceAttrSet("tencentcloud_clb_instance.default_enable", "security_groups.0"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.default_enable", "target_region_info_region", "ap-guangzhou"),
					resource.TestCheckResourceAttrSet("tencentcloud_clb_instance.default_enable", "target_region_info_vpc_id"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.default_enable", "tags.test", "open"),
				),
			},
			{
				Config: testAccClbInstance_default_enable_open,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClbInstanceExists("tencentcloud_clb_instance.default_enable"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.default_enable", "network_type", "OPEN"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.default_enable", "clb_name", "my_open_clb"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.default_enable", "project_id", "0"),
					resource.TestCheckResourceAttrSet("tencentcloud_clb_instance.default_enable", "vpc_id"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.default_enable", "load_balancer_pass_to_target", "true"),
					resource.TestCheckResourceAttrSet("tencentcloud_clb_instance.default_enable", "security_groups.0"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.default_enable", "target_region_info_region", "ap-guangzhou"),
					resource.TestCheckResourceAttrSet("tencentcloud_clb_instance.default_enable", "target_region_info_vpc_id"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.default_enable", "tags.test", "hello"),
				),
			},
		},
	})
}

func TestAccTencentCloudClbInstance_multiple_instance(t *testing.T) {
	t.Parallel()

	resource.Test(t, resource.TestCase{
		PreCheck:     func() { testAccPreCheck(t) },
		Providers:    testAccProviders,
		CheckDestroy: testAccCheckClbInstanceDestroy,
		Steps: []resource.TestStep{
			{
				Config: testAccClbInstance__multi_instance,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClbInstanceExists("tencentcloud_clb_instance.multiple_instance"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.multiple_instance", "network_type", "OPEN"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.multiple_instance", "clb_name", "my_open_clb"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.multiple_instance", "master_zone_id", "10001"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.multiple_instance", "slave_zone_id", "10002"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.multiple_instance", "tags.test", "mytest"),
				),
			},
			{
				Config: testAccClbInstance__multi_instance_update,
				Check: resource.ComposeTestCheckFunc(
					testAccCheckClbInstanceExists("tencentcloud_clb_instance.multiple_instance"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.multiple_instance", "network_type", "OPEN"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.multiple_instance", "clb_name", "my_open_clb"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.multiple_instance", "master_zone_id", "10001"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.multiple_instance", "slave_zone_id", "10002"),
					resource.TestCheckResourceAttr("tencentcloud_clb_instance.multiple_instance", "tags.test", "open"),
				),
			},
		},
	})
}

func testAccCheckClbInstanceDestroy(s *terraform.State) error {
	logId := getLogId(contextNil)
	ctx := context.WithValue(context.TODO(), logIdKey, logId)

	clbService := ClbService{
		client: testAccProvider.Meta().(*TencentCloudClient).apiV3Conn,
	}
	for _, rs := range s.RootModule().Resources {
		if rs.Type != "tencentcloud_clb_instance" {
			continue
		}

		instance, err := clbService.DescribeLoadBalancerById(ctx, rs.Primary.ID)
		if instance != nil && err == nil {
			return fmt.Errorf("[CHECK][CLB instance][Destroy] check: CLB instance still exists: %s", rs.Primary.ID)
		}
	}
	return nil
}

func testAccCheckClbInstanceExists(n string) resource.TestCheckFunc {
	return func(s *terraform.State) error {
		logId := getLogId(contextNil)
		ctx := context.WithValue(context.TODO(), logIdKey, logId)

		rs, ok := s.RootModule().Resources[n]
		if !ok {
			return fmt.Errorf("[CHECK][CLB instance][Exists] check: CLB instance %s is not found", n)
		}
		if rs.Primary.ID == "" {
			return fmt.Errorf("[CHECK][CLB instance][Exists] check: CLB instance id is not set")
		}
		clbService := ClbService{
			client: testAccProvider.Meta().(*TencentCloudClient).apiV3Conn,
		}
		instance, err := clbService.DescribeLoadBalancerById(ctx, rs.Primary.ID)
		if err != nil {
			return err
		}
		if instance == nil {
			return fmt.Errorf("[CHECK][CLB instance][Exists] id %s is not exist", rs.Primary.ID)
		}
		return nil
	}
}

const testAccClbInstance_basic = `
resource "tencentcloud_clb_instance" "clb_basic" {
  network_type = "OPEN"
  clb_name     = "tf-clb-basic"
}
`

const testAccClbInstance_internal = `
variable "availability_zone" {
  default = "ap-guangzhou-3"
}

resource "tencentcloud_vpc" "foo" {
  name       = "guagua-ci-temp-test"
  cidr_block = "10.0.0.0/16"
}

resource "tencentcloud_subnet" "subnet" {
  availability_zone = var.availability_zone
  name              = "guagua-ci-temp-test"
  vpc_id            = tencentcloud_vpc.foo.id
  cidr_block        = "10.0.20.0/28"
  is_multicast      = false
}

resource "tencentcloud_clb_instance" "clb_internal" {
  network_type = "INTERNAL"
  clb_name     = "tf-clb-internal"
  vpc_id       = tencentcloud_vpc.foo.id
  subnet_id    = tencentcloud_subnet.subnet.id
  project_id   = 0

  tags = {
    test = "tf1"
  }
}
`

const testAccClbInstance_open = `
resource "tencentcloud_security_group" "foo" {
  name = "ci-temp-test-sg"
}

resource "tencentcloud_vpc" "foo" {
  name       = "guagua-ci-temp-test"
  cidr_block = "10.0.0.0/16"
}

resource "tencentcloud_clb_instance" "clb_open" {
  network_type              = "OPEN"
  clb_name                  = "tf-clb-open"
  project_id                = 0
  vpc_id                    = tencentcloud_vpc.foo.id
  target_region_info_region = "ap-guangzhou"
  target_region_info_vpc_id = tencentcloud_vpc.foo.id
  security_groups           = [tencentcloud_security_group.foo.id]

  tags = {
    test = "tf"
  }
}
`

const testAccClbInstance_update = `
variable "availability_zone" {
  default = "ap-guangzhou-3"
}

resource "tencentcloud_vpc" "foo" {
  name       = "guagua-ci-temp-test"
  cidr_block = "10.0.0.0/16"
}

resource "tencentcloud_subnet" "subnet" {
  availability_zone = var.availability_zone
  name              = "guagua-ci-temp-test"
  vpc_id            = tencentcloud_vpc.foo.id
  cidr_block        = "10.0.20.0/28"
  is_multicast      = false
}

resource "tencentcloud_clb_instance" "clb_internal" {
  network_type = "INTERNAL"
  clb_name     = "tf-clb-update-internal"
  vpc_id       = tencentcloud_vpc.foo.id
  subnet_id    = tencentcloud_subnet.subnet.id
  project_id   = 0

  tags = {
    test = "test"
  }
}
`

const testAccClbInstance_update_open = `
resource "tencentcloud_security_group" "foo" {
  name = "ci-temp-test1-sg"
}

resource "tencentcloud_vpc" "foo" {
  name       = "guagua-ci-temp-test"
  cidr_block = "10.0.0.0/16"
}

resource "tencentcloud_clb_instance" "clb_open" {
  network_type              = "OPEN"
  clb_name                  = "tf-clb-update-open"
  vpc_id                    = tencentcloud_vpc.foo.id
  project_id                = 0
  target_region_info_region = "ap-guangzhou"
  target_region_info_vpc_id = tencentcloud_vpc.foo.id
  security_groups           = [tencentcloud_security_group.foo.id]

  tags = {
    test = "test"
  }
}
`

const testAccClbInstance_default_enable = `
variable "availability_zone" {
  default = "ap-guangzhou-1"
}

resource "tencentcloud_subnet" "subnet" {
  availability_zone = var.availability_zone
  name              = "sdk-feature-test"
  vpc_id            = tencentcloud_vpc.foo.id
  cidr_block        = "10.0.20.0/28"
  is_multicast      = false
}

resource "tencentcloud_security_group" "sglab" {
  name        = "sg_o0ek7r93"
  description = "favourite sg"
  project_id  = 0
}

resource "tencentcloud_vpc" "foo" {
  name         = "for-my-open-clb"
  cidr_block   = "10.0.0.0/16"

  tags = {
    "test" = "mytest"
  }
}

resource "tencentcloud_clb_instance" "default_enable" {
  network_type                 = "OPEN"
  clb_name                     = "my-open-clb"
  project_id                   = 0
  vpc_id                       = tencentcloud_vpc.foo.id
  load_balancer_pass_to_target = true

  security_groups              = [tencentcloud_security_group.sglab.id]
  target_region_info_region    = "ap-guangzhou"
  target_region_info_vpc_id    = tencentcloud_vpc.foo.id

  tags = {
    test = "open"
  }
}
`

const testAccClbInstance_default_enable_open = `
variable "availability_zone" {
  default = "ap-guangzhou-1"
}

resource "tencentcloud_subnet" "subnet" {
  availability_zone = var.availability_zone
  name              = "sdk-feature-test"
  vpc_id            = tencentcloud_vpc.foo.id
  cidr_block        = "10.0.20.0/28"
  is_multicast      = false
}

resource "tencentcloud_security_group" "sglab" {
  name        = "sg_o0ek7r93"
  description = "favourite sg"
  project_id  = 0
}

resource "tencentcloud_vpc" "foo" {
  name         = "for-my-open-clb"
  cidr_block   = "10.0.0.0/16"

  tags = {
    "test" = "mytest"
  }
}

resource "tencentcloud_clb_instance" "default_enable" {
  network_type                 = "OPEN"
  clb_name                     = "my-open-clb"
  project_id                   = 0
  vpc_id                       = tencentcloud_vpc.foo.id
  load_balancer_pass_to_target = true

  security_groups              = [tencentcloud_security_group.sglab.id]
  target_region_info_region    = "ap-guangzhou"
  target_region_info_vpc_id    = tencentcloud_vpc.foo.id

  tags = {
    test = "hello"
  }
}
`

const testAccClbInstance__multi_instance = `
resource "tencentcloud_clb_instance" "multiple_instance" {
  network_type              = "OPEN"
  clb_name                  = "my-open-clb"
  master_zone_id = "10001"
  slave_zone_id = "10002"

  tags = {
    test = "open"
  }
}
`

const testAccClbInstance__multi_instance_update = `
resource "tencentcloud_clb_instance" "multiple_instance" {
  network_type              = "OPEN"
  clb_name                  = "my-open-clb"
  master_zone_id = "10001"
  slave_zone_id = "10002"

  tags = {
    test = "open"
  }
}
`
